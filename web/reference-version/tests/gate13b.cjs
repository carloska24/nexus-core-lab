const { chromium } = require('C:/Users/joaob/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const origin = process.env.GATE13B_ORIGIN || 'http://127.0.0.1:5193';
const output = path.resolve(__dirname, '../evidence/gate13b');
const headed = process.env.GATE13B_HEADED === '1';
fs.mkdirSync(output, { recursive: true });
const results = [], errors = [], consoleErrors = [], hero = [], layouts = [];
const pass = message => { results.push(message); console.log('PASS', message); };
const until = async (check, message) => {
  for (let index = 0; index < 120; index += 1) {
    if (await check()) return;
    await new Promise(resolve => setTimeout(resolve, 200));
  }
  throw new Error(`Timeout: ${message}`);
};

(async () => {
  const browser = await chromium.launch({ channel: 'chrome', headless: !headed });
  const visitorA = await browser.newContext();
  const visitorB = await browser.newContext();
  const page = await visitorA.newPage();
  const isolated = await visitorB.newPage();
  page.on('pageerror', error => errors.push(`pageerror: ${error.message}`));
  page.on('console', message => { if (message.type() === 'error') consoleErrors.push(message.text()); });
  const nav = async route => { await page.locator(`a[href="#${route}"]`).first().click(); await page.waitForTimeout(120); };
  const button = name => page.getByRole('button', { name, exact: true });
  const mapReady = async target => target.waitForFunction(() => document.querySelector('.map-experience')?.getAttribute('data-map-mode') === 'ready', { timeout: 30000 });
  const topologySuccess = async target => target.waitForFunction(() => document.querySelector('[data-testid="topology-state"]')?.getAttribute('data-state') === 'success', { timeout: 30000 });
  try {
    // Establish one visitor cookie before the application's parallel initial reads.
    assert.ok((await visitorA.request.get(`${origin}/api/v1/devices`)).ok());
    assert.ok((await visitorB.request.get(`${origin}/api/v1/devices`)).ok());
    await page.goto(`${origin}/#network`);
    await mapReady(page);
    await topologySuccess(page);
    assert.match(await page.getByTestId('map-truth').innerText(), /Simulated telecom topology over real Campinas cartography/);
    assert.match(await page.locator('.maplibregl-ctrl-attrib').innerText(), /OpenFreeMap/);
    assert.equal(await page.getByTestId('map-device').count(), 0);
    pass('Real Campinas cartography loads with honest simulated-topology label and visible attribution');

    for (const [width, height] of [[1920, 1080], [1440, 900], [1366, 768]]) {
      await page.setViewportSize({ width, height });
      await page.waitForTimeout(350);
      const layout = await page.locator('.network-page').evaluate((element, size) => ({
        ...size, clientWidth: element.clientWidth, scrollWidth: element.scrollWidth,
        clientHeight: element.clientHeight, scrollHeight: element.scrollHeight,
      }), { width, height });
      layouts.push(layout);
      assert.equal(layout.scrollWidth, layout.clientWidth, `Network horizontal clipping at ${width}x${height}`);
      assert.equal(layout.scrollHeight, layout.clientHeight, `Network vertical clipping at ${width}x${height}`);
      await page.screenshot({ path: path.join(output, `network-empty-${width}x${height}.png`) });
    }
    pass('Network has no clipping or overflow at 1920x1080, 1440x900 and 1366x768');

    await isolated.goto(`${origin}/#network`);
    await mapReady(isolated);
    await topologySuccess(isolated);
    assert.equal(await isolated.getByTestId('map-device').count(), 0);

    const suffix = String(Date.now()).slice(-9);
    const imsi = `724059${suffix}`;
    const imei = `860010${suffix}`;
    await nav('subscribers');
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
    await nav('network');
    await until(async () => await page.getByTestId('map-device').count() === 1, 'map attach representation');
    await mapReady(page);
    const attached = await page.getByTestId('map-device').evaluate(element => ({
      device: element.dataset.deviceId, session: element.dataset.sessionId, cell: element.dataset.cellId, ip: element.dataset.ip,
    }));
    assert.equal(attached.cell, 'CELL-SP-001');
    assert.ok(attached.session && attached.ip);
    await page.getByRole('button', { name: 'Topology device states' }).click();
    assert.match(await page.locator('.topology-register').innerText(), new RegExp(attached.ip.replaceAll('.', '\\.')));
    await page.waitForTimeout(1500);
    await page.screenshot({ path: path.join(output, 'hero-attach-cell-001.png') });
    hero.push({ action: 'Attach', ...attached });
    pass('Attach renders the authoritative connected Session, Cell and inspectable IP');

    await nav('sessions');
    await page.getByLabel('Session device', { exact: true }).selectOption(deviceID);
    await page.getByLabel('Session cell', { exact: true }).selectOption('CELL-SP-002');
    await button('Handover').click();
    await until(async () => /Server confirmed handover/.test(await page.getByTestId('session-panel').innerText()), 'handover confirmation');
    await nav('network');
    await until(async () => await page.getByTestId('map-device').getAttribute('data-cell-id') === 'CELL-SP-002', 'map handover representation');
    await mapReady(page);
    const handed = await page.getByTestId('map-device').evaluate(element => ({ session: element.dataset.sessionId, cell: element.dataset.cellId, ip: element.dataset.ip }));
    assert.equal(handed.session, attached.session);
    assert.equal(handed.ip, attached.ip);
    await page.waitForTimeout(1500);
    await page.screenshot({ path: path.join(output, 'hero-handover-cell-002.png') });
    hero.push({ action: 'Handover', ...handed });
    await nav('events');
    await until(async () => /CELL_HANDOVER/.test(await page.locator('main').innerText()), 'handover event');
    pass('Handover moves the association while preserving Session ID and IP; event remains visible');

    await nav('sessions');
    await page.getByLabel('Session device', { exact: true }).selectOption(deviceID);
    await button('Detach').click();
    await until(async () => /DISCONNECTED/.test(await page.getByTestId('session-panel').innerText()), 'detach confirmation');
    await nav('network');
    await until(async () => await page.getByTestId('map-device').count() === 0, 'map detach representation');
    await nav('telemetry');
    await until(async () => await page.getByTestId('pool-allocated').innerText() === '0', 'IP pool release');
    await nav('events');
    await until(async () => /DETACH/.test(await page.locator('main').innerText()), 'detach event');
    hero.push({ action: 'Detach', session: attached.session, releasedIP: attached.ip });
    pass('Detach removes the map association, releases the IP and leaves the event evidence');

    await isolated.reload();
    await mapReady(isolated);
    await topologySuccess(isolated);
    assert.equal(await isolated.getByTestId('map-device').count(), 0);
    assert.match(await isolated.getByTestId('topology-state').innerText(), /0 connected · 0 without session/);
    pass('A second public-demo visitor remains isolated from the complete Hero Flow');

    const fallbackContext = await browser.newContext();
    const fallback = await fallbackContext.newPage();
    assert.ok((await fallbackContext.request.get(`${origin}/api/v1/devices`)).ok());
    await fallback.route('https://tiles.openfreemap.org/**', route => route.abort('failed'));
    await fallback.goto(`${origin}/#network`);
    await fallback.getByTestId('map-fallback').waitFor({ timeout: 30000 });
    assert.equal(await fallback.locator('.fallback-map .tower').count(), 3);
    assert.match(await fallback.getByTestId('map-fallback').innerText(), /Local map fallback/);
    assert.match(await fallback.getByTestId('map-truth').innerText(), /Simulated telecom topology/);
    await fallback.screenshot({ path: path.join(output, 'provider-fallback.png') });
    await fallbackContext.close();
    pass('Provider failure selects the local map; all three Cells and the live-state layer remain available');

    assert.equal(await page.locator('.maplibregl-ctrl-geolocate').count(), 0);
    assert.deepEqual(errors, []);
    assert.ok(consoleErrors.every(message => /404 \(Not Found\)/.test(message)), `Unexpected console error: ${consoleErrors.join(' | ')}`);
    pass('No browser geolocation behavior and zero uncaught JavaScript exceptions in the normal flow');
  } finally {
    fs.writeFileSync(path.join(output, 'results.json'), JSON.stringify({ results, errors, expectedHTTP404ConsoleMessages: consoleErrors, hero, layouts, headed }, null, 2));
    await visitorA.close();
    await visitorB.close();
    await browser.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
