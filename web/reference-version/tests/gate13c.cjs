const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const playwrightModule = process.env.PLAYWRIGHT_MODULE || 'playwright';
const { chromium } = require(playwrightModule);
const origin = process.env.GATE13C_ORIGIN || 'http://127.0.0.1:18083';
const output = path.resolve(__dirname, '../evidence/gate13c');
fs.mkdirSync(output, { recursive: true });

const results = [];
const errors = [];
const consoleErrors = [];
const hero = [];
const layouts = [];
const pass = message => { results.push(message); console.log('PASS', message); };
const until = async (check, message, attempts = 150) => {
  for (let index = 0; index < attempts; index += 1) {
    if (await check()) return;
    await new Promise(resolve => setTimeout(resolve, 200));
  }
  throw new Error(`Timeout: ${message}`);
};

(async () => {
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  const visitorA = await browser.newContext();
  const visitorB = await browser.newContext();
  const page = await visitorA.newPage();
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => { if (message.type() === 'error') consoleErrors.push(message.text()); });
  const nav = async route => { await page.locator(`a[href="#${route}"]`).first().click(); await page.waitForTimeout(150); };
  const button = name => page.getByRole('button', { name, exact: true });
  try {
    const root = await visitorA.request.get(`${origin}/`);
    assert.ok(root.ok());
    assert.equal(root.headers()['cache-control'], 'no-cache');
    assert.match(root.headers()['content-security-policy'], /default-src 'self'/);
    assert.equal(root.headers()['permissions-policy'], 'geolocation=(), camera=(), microphone=()');
    const html = await root.text();
    const assetPath = html.match(/src="([^"]+\.js)"/)?.[1];
    assert.ok(assetPath, 'compiled JavaScript asset is referenced');
    const asset = await visitorA.request.get(new URL(assetPath, origin).toString());
    assert.equal(asset.headers()['cache-control'], 'public, max-age=31536000, immutable');
    const api404 = await visitorA.request.get(`${origin}/api/v1/does-not-exist`);
    assert.equal(api404.status(), 404);
    assert.doesNotMatch(await api404.text(), /<html/i);
    pass('Go serves SPA and immutable assets without masking API 404 responses');

    const health = await visitorA.request.get(`${origin}/health`);
    assert.deepEqual(await health.json(), { status: 'ok', service: 'nexus-core-lab' });
    const devicesA = await visitorA.request.get(`${origin}/api/v1/devices`);
    assert.ok(devicesA.ok());
    const cookie = (await visitorA.cookies()).find(item => item.name === 'nexus_demo_session');
    assert.ok(cookie?.httpOnly);
    assert.equal(cookie.sameSite, 'Strict');
    pass('Health, anonymous HttpOnly visitor cookie and SameSite isolation are active');

    const suffix = String(Date.now()).slice(-9);
    const imsi = `724059${suffix}`;
    const imei = `860010${suffix}`;
    await page.goto(`${origin}/#subscribers`);
    await button('Provision subscriber').click();
    await page.getByLabel('Provision IMSI', { exact: true }).fill(imsi);
    await page.getByLabel('Provision MSISDN', { exact: true }).fill(`19${suffix}`);
    await button('Provision').click();
    await button('Activate').click();
    await until(async () => /Server confirmed: ACTIVE/.test(await page.locator('dialog').innerText()), 'subscriber activation');
    await page.getByLabel('Close subscriber dialog').click();
    hero.push({ action: 'Provision + Activate', imsi });

    await nav('devices');
    await button('Register device').click();
    const subscriber = await page.getByLabel('Device subscriber', { exact: true }).locator('option').filter({ hasText: imsi }).getAttribute('value');
    await page.getByLabel('Device subscriber', { exact: true }).selectOption(subscriber);
    await page.getByLabel('Register IMEI', { exact: true }).fill(imei);
    await button('Register').click();
    await until(async () => /Device registered/.test(await page.locator('dialog').innerText()), 'device registration');
    await page.getByLabel('Close device dialog').click();
    hero.push({ action: 'Register', imei });

    await nav('sessions');
    const deviceID = await page.getByLabel('Session device', { exact: true }).locator('option').filter({ hasText: imei }).getAttribute('value');
    await page.getByLabel('Session device', { exact: true }).selectOption(deviceID);
    await page.getByLabel('Session cell', { exact: true }).selectOption('CELL-SP-001');
    await button('Attach').click();
    await button('Detach').waitFor();
    const attachFields = await page.getByTestId('session-details').innerText();
    const sessionID = attachFields.match(/Session UUID\s+([^\s]+)/)?.[1];
    const ip = attachFields.match(/IP address\s+([^\s]+)/)?.[1];
    assert.ok(sessionID && ip);
    hero.push({ action: 'Attach', sessionID, ip, cell: 'CELL-SP-001' });

    await page.getByLabel('Session cell', { exact: true }).selectOption('CELL-SP-002');
    await button('Handover').click();
    await until(async () => /Server confirmed handover/.test(await page.getByTestId('session-panel').innerText()), 'handover');
    const handoverFields = await page.getByTestId('session-details').innerText();
    assert.match(handoverFields, new RegExp(sessionID.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
    assert.match(handoverFields, new RegExp(ip.replaceAll('.', '\\.')));
    assert.match(handoverFields, /CELL-SP-002/);
    hero.push({ action: 'Handover', sessionID, ip, cell: 'CELL-SP-002' });

    await nav('network');
    await page.getByTestId('topology-state').waitFor({ timeout: 30000 });
    await until(async () => await page.getByTestId('map-device').getAttribute('data-cell-id') === 'CELL-SP-002', 'map handover state');
    await until(async () => ['ready', 'fallback'].includes(await page.locator('.map-experience').getAttribute('data-map-mode')), 'map initialization');
    const mapMode = await page.locator('.map-experience').getAttribute('data-map-mode');
    assert.ok(['ready', 'fallback'].includes(mapMode));
    if (mapMode === 'ready') assert.match(await page.getByTestId('map-truth').innerText(), /real Campinas cartography/i);
    await page.screenshot({ path: path.join(output, 'hero-handover.png') });
    pass('Hero Flow reaches the Campinas topology with authoritative Session, Cell and IP');

    await nav('sessions');
    await page.getByLabel('Session device', { exact: true }).selectOption(deviceID);
    await button('Detach').click();
    await until(async () => /DISCONNECTED/.test(await page.getByTestId('session-panel').innerText()), 'detach');
    await nav('telemetry');
    await until(async () => await page.getByTestId('pool-allocated').innerText() === '0', 'IP release');
    await nav('events');
    await until(async () => /DETACH/.test(await page.locator('main').innerText()), 'detach event');
    hero.push({ action: 'Detach', sessionID, releasedIP: ip });
    pass('Detach releases the IP and exposes the authoritative event');

    const suffixB = String(Date.now() + 1).slice(-9);
    const createB = await visitorB.request.post(`${origin}/api/v1/subscribers`, {
      headers: { 'Content-Type': 'application/json', Origin: origin },
      data: { imsi: `724058${suffixB}`, msisdn: `18${suffixB}` },
    });
    assert.equal(createB.status(), 201);
    assert.equal((await (await visitorB.request.get(`${origin}/api/v1/subscribers`)).json()).length, 1);
    assert.equal((await (await visitorA.request.get(`${origin}/api/v1/subscribers`)).json()).length, 1);
    pass('Two visitor contexts retain independent telecom records');

    await nav('configuration');
    await button('Reset Demo').click();
    await button('Confirm Reset Demo').click();
    await page.waitForURL(/#overview$/, { timeout: 10000 });
    assert.equal((await (await visitorA.request.get(`${origin}/api/v1/subscribers`)).json()).length, 0);
    assert.equal((await (await visitorB.request.get(`${origin}/api/v1/subscribers`)).json()).length, 1);
    pass('Reset Demo clears only the requesting visitor and reloads a clean dashboard');

    for (const [width, height] of [[1920, 1080], [1440, 900], [1366, 768]]) {
      await page.setViewportSize({ width, height });
      await page.goto(`${origin}/#network`);
      await page.waitForTimeout(700);
      const layout = await page.locator('.network-page').evaluate((element, size) => ({
        ...size, clientWidth: element.clientWidth, scrollWidth: element.scrollWidth,
        clientHeight: element.clientHeight, scrollHeight: element.scrollHeight,
      }), { width, height });
      layouts.push(layout);
      assert.equal(layout.scrollWidth, layout.clientWidth);
      assert.equal(layout.scrollHeight, layout.clientHeight);
      await page.screenshot({ path: path.join(output, `network-${width}x${height}.png`) });
    }
    pass('Production container layout fits 1920x1080, 1440x900 and 1366x768');

    const fallbackContext = await browser.newContext();
    const fallback = await fallbackContext.newPage();
    await fallbackContext.request.get(`${origin}/api/v1/devices`);
    await fallback.route('https://tiles.openfreemap.org/**', route => route.abort('failed'));
    await fallback.goto(`${origin}/#network`);
    await fallback.getByTestId('map-fallback').waitFor({ timeout: 30000 });
    assert.equal(await fallback.locator('.fallback-map .tower').count(), 3);
    await fallbackContext.close();
    pass('OpenFreeMap failure selects the local topology fallback');

    assert.deepEqual(errors, []);
    assert.ok(consoleErrors.every(message => /404 \(Not Found\)/.test(message)), `Unexpected console error: ${consoleErrors.join(' | ')}`);
    pass('Zero uncaught JavaScript exceptions in the production-like flow');
  } finally {
    fs.writeFileSync(path.join(output, 'results.json'), JSON.stringify({ origin, results, errors, consoleErrors, hero, layouts }, null, 2));
    await visitorA.close();
    await visitorB.close();
    await browser.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
