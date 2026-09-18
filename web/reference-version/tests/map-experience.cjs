const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const origin = process.env.MAP_EXPERIENCE_ORIGIN || 'http://127.0.0.1:18084';
const output = path.resolve(__dirname, '../evidence/map-experience');
fs.mkdirSync(output, { recursive: true });
const viewports = [[1920, 1080], [1440, 900], [1366, 768]];
const results = { origin, checks: [], transport: { workers: [], tiles: [], failures: [] }, states: {}, pageErrors: [], consoleErrors: [] };
const pass = message => { results.checks.push(message); console.log('PASS', message); };
const until = async (check, message, attempts = 180) => {
  for (let index = 0; index < attempts; index += 1) {
    if (await check()) return;
    await new Promise(resolve => setTimeout(resolve, 200));
  }
  throw new Error('Timeout: ' + message);
};

(async () => {
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();
  page.on('pageerror', error => results.pageErrors.push(error.message));
  page.on('console', message => { if (message.type() === 'error') results.consoleErrors.push(message.text()); });
  page.on('requestfailed', request => results.transport.failures.push({ url: request.url(), error: request.failure()?.errorText }));
  page.on('response', response => {
    const url = response.url();
    if (/maplibre-gl-worker/.test(url)) results.transport.workers.push({ url, status: response.status() });
    if (/\.pbf(?:\?|$)/.test(url) && /openfreemap/.test(url)) results.transport.tiles.push({ url, status: response.status() });
  });
  const headers = { Origin: origin, 'Content-Type': 'application/json' };
  const post = async (pathname, data) => {
    const response = await context.request.post(origin + pathname, { headers, data });
    assert.ok(response.ok(), pathname + ' returned ' + response.status() + ': ' + await response.text());
    return response.json();
  };
  const capture = async (state, target = page) => {
    const layouts = [];
    for (const [width, height] of viewports) {
      await target.setViewportSize({ width, height });
      await target.waitForTimeout(350);
      const layout = await target.locator('.network-page').evaluate((element, viewport) => ({
        viewport, clientWidth: element.clientWidth, scrollWidth: element.scrollWidth,
        clientHeight: element.clientHeight, scrollHeight: element.scrollHeight,
      }), { width, height });
      assert.equal(layout.scrollWidth, layout.clientWidth, state + ' horizontal overflow at ' + width);
      assert.equal(layout.scrollHeight, layout.clientHeight, state + ' vertical overflow at ' + height);
      layouts.push(layout);
      await target.screenshot({ path: path.join(output, state + '-' + width + 'x' + height + '.png') });
    }
    results.states[state] = layouts;
  };

  try {
    await page.goto(origin + '/#network');
    await page.getByTestId('topology-state').waitFor({ timeout: 30000 });
    await until(async () => await page.locator('.map-experience').getAttribute('data-map-mode') === 'ready', 'initial map readiness');
    await until(() => results.transport.tiles.some(item => item.status === 200), 'first OpenFreeMap vector tile');
    assert.ok(results.transport.workers.some(item => item.status === 200), 'bundled MapLibre worker must return HTTP 200');
    assert.equal(results.transport.failures.filter(item => /maplibre-gl-(?:worker|shared)/.test(item.url)).length, 0);
    await page.getByTestId('map-empty').waitFor();
    await capture('empty');
    pass('Empty topology is intentional and MapLibre worker plus real Campinas tiles load successfully');

    const suffix = String(Date.now()).slice(-9);
    const subscriber = await post('/api/v1/subscribers', { imsi: '724059' + suffix, msisdn: '19' + suffix });
    await post('/api/v1/subscribers/' + encodeURIComponent(subscriber.id) + '/activate');
    const device = await post('/api/v1/devices', { subscriber_id: subscriber.id, imei: '860010' + suffix, technology: '5G' });
    const session = await post('/api/v1/sessions/attach', { device_id: device.id, cell_id: 'CELL-SP-001' });

    await page.reload();
    await page.getByTestId('topology-state').waitFor({ timeout: 30000 });
    await until(async () => await page.getByTestId('map-device').getAttribute('data-cell-id') === 'CELL-SP-001', 'attached device on CELL-SP-001');
    await until(async () => await page.locator('.map-experience').getAttribute('data-map-mode') === 'ready', 'attached map readiness');
    await page.waitForTimeout(900);
    await capture('attached');
    pass('Attached state shows a deterministic device, serving Cell and logical link');

    const handed = await post('/api/v1/sessions/' + encodeURIComponent(session.id) + '/handover', { target_cell_id: 'CELL-SP-002' });
    assert.equal(handed.id, session.id);
    assert.equal(handed.ip_address, session.ip_address);
    await until(async () => await page.getByTestId('map-device').getAttribute('data-cell-id') === 'CELL-SP-002', 'handover topology refresh');
    await page.getByTestId('handover-feedback').waitFor({ timeout: 30000 });
    assert.match(await page.getByTestId('handover-feedback').innerText(), /CELL-SP-001[\s\S]*CELL-SP-002/);
    await capture('handover');
    pass('Handover preserves Session and IP while visual association moves to CELL-SP-002');

    await post('/api/v1/sessions/' + encodeURIComponent(session.id) + '/detach', {});
    await until(async () => await page.getByTestId('map-device').count() === 0, 'detached topology refresh');
    await page.getByTestId('map-empty').waitFor();
    await capture('detached');
    pass('Detached state removes the dynamic device and logical link');

    const fallbackContext = await browser.newContext();
    const fallback = await fallbackContext.newPage();
    await fallback.route('https://tiles.openfreemap.org/**', route => route.abort('failed'));
    await fallback.goto(origin + '/#network');
    await fallback.getByTestId('map-fallback').waitFor({ timeout: 30000 });
    assert.equal(await fallback.locator('.fallback-map .tower').count(), 3);
    await capture('fallback', fallback);
    await fallbackContext.close();
    pass('Provider failure preserves the local, truthful topology fallback');

    assert.deepEqual(results.pageErrors, []);
    assert.ok(results.consoleErrors.every(message => /404 \(Not Found\)/.test(message)), 'Unexpected console error: ' + results.consoleErrors.join(' | '));
    assert.ok(results.transport.tiles.every(item => item.status === 200));
    pass('Zero uncaught JavaScript exceptions; console output contains only expected active-session 404 responses');
  } finally {
    try { await context.request.post(origin + '/api/v1/demo/reset', { headers }); } catch {}
    fs.writeFileSync(path.join(output, 'results.json'), JSON.stringify(results, null, 2));
    await context.close();
    await browser.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
