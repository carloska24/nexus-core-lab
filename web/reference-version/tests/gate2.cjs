// Run against the explicitly isolated in-memory API on :18080 and Vite on :5176.
// Uses the desktop's existing Playwright runtime, no project dependency added.
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'C:/Users/joaob/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const output = path.resolve(__dirname, '../evidence/gate2');
fs.mkdirSync(output, { recursive: true });
const results = [];
const origin = 'http://127.0.0.1:5176';
const api = 'http://127.0.0.1:18080';
const passed = name => { results.push(name); console.log('PASS:', name); };
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

(async () => {
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  const page = await browser.newPage({ viewport: { width: 1920, height: 1080 }, deviceScaleFactor: 1 });
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const state = async (id, value) => page.waitForFunction(({id,value}) => document.querySelector(`[data-testid="${id}"]`)?.getAttribute('data-state') === value, {id,value}, { timeout: 25000 });
  const value = async (id, expected) => page.waitForFunction(({id,expected}) => document.querySelector(`[data-testid="${id}"] strong`)?.textContent === String(expected), {id,expected}, { timeout: 25000 });
  try {
    // Initial failures must not become zeroes or green health.
    await page.route('**/health', route => route.abort('connectionfailed'));
    await page.route('**/telemetry', route => route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({error:'internal_error',message:'test storage failure',code:'TELEMETRY_STORAGE_ERROR'}) }));
    await page.goto(origin);
    await state('active-kpi', 'error');
    assert.equal(await page.locator('[data-testid="active-kpi"] strong').innerText(), '—');
    await page.waitForFunction(() => document.querySelector('[data-testid="system-status"]')?.textContent.includes('OFFLINE'));
    assert.equal(await page.locator('.infra i[data-status="unknown"]').count(), 1);
    passed('Initial network/telemetry failures: unavailable, no zero fallback, DB unknown');
    await page.unroute('**/health');
    await page.unroute('**/telemetry');
    await state('active-kpi', 'success');
    await page.waitForFunction(() => document.querySelector('[data-testid="system-status"]')?.textContent.includes('ONLINE'), null, {timeout:25000});
    const before = await (await fetch(`${api}/telemetry`)).json();
    const sum = t => Object.values(t.events_total).reduce((a,b)=>a+b,0);
    await value('active-kpi', before.metrics.active_sessions);
    await value('events-kpi', sum(before));
    passed('Real Go /health and /telemetry via Vite proxy match rendered KPIs');
    const write = async (route, body) => {
      const response = await fetch(api + route, {method:'POST', headers:{'Content-Type':'application/json'}, body: body === undefined ? undefined : JSON.stringify(body)});
      assert.ok(response.ok, `${route}: ${response.status}`);
      return response.json();
    };
    const suffix = String(Date.now()).slice(-10);
    const sub = await write('/api/v1/subscribers', {imsi:'72499'+suffix,msisdn:'+5519'+suffix});
    await write(`/api/v1/subscribers/${sub.id}/activate`);
    const device = await write('/api/v1/devices',{subscriber_id:sub.id,imei:'86000'+suffix,technology:'5G'});
    const session = await write('/api/v1/sessions/attach',{device_id:device.id,cell_id:'CELL-SP-001'});
    await value('active-kpi', before.metrics.active_sessions + 1);
    await value('events-kpi', sum(before) + 1);
    const attached = await (await fetch(`${api}/telemetry`)).json();
    await page.waitForFunction(() => Number(document.querySelector('[data-testid="activity"] svg.line-chart')?.getAttribute('aria-label')?.match(/(\d+) samples/)?.[1]) >= 2, null, { timeout: 20000 });
    await page.screenshot({path:path.join(output,'1920x1080-online.png')});
    passed('Real provision → activate → register → attach observed by UI: +1 active, +1 event');
    await write(`/api/v1/sessions/${session.id}/handover`,{target_cell_id:'CELL-SP-002'});
    await write(`/api/v1/sessions/${session.id}/detach`);
    await value('active-kpi',before.metrics.active_sessions);
    await value('events-kpi',sum(before)+3);
    const after = await (await fetch(`${api}/telemetry`)).json();
    fs.writeFileSync(path.join(output,'real-telemetry.json'),JSON.stringify({before,attached,after,session_id:session.id},null,2));
    passed('Real handover/detach observed: active returns to baseline; event total +3');
    const last = await page.locator('[data-testid="events-kpi"] strong').innerText();
    await page.route('**/telemetry', route => route.fulfill({status:500,contentType:'application/json',body:'{"code":"TELEMETRY_STORAGE_ERROR","message":"test storage failure","error":"internal_error"}'}));
    await state('events-kpi','stale');
    assert.equal(await page.locator('[data-testid="events-kpi"] strong').innerText(),last);
    assert.match(await page.locator('[data-testid="system-status"]').innerText(),/ONLINE/);
    await page.screenshot({path:path.join(output,'1920x1080-telemetry-stale.png')});
    passed('Telemetry 500 keeps last snapshot marked stale; independent health stays online');
    await page.unroute('**/telemetry');
    await page.route('**/telemetry', route => route.abort('connectionfailed'));
    await page.route('**/health', route => route.abort('connectionfailed'));
    await page.waitForFunction(() => document.querySelector('[data-testid="system-status"]')?.textContent.includes('OFFLINE'),null,{timeout:25000});
    assert.equal(await page.locator('[data-testid="events-kpi"] strong').innerText(),last);
    await page.screenshot({path:path.join(output,'1920x1080-offline.png')});
    passed('Network failure after success: offline health, stale telemetry retains last values');
    await page.unroute('**/telemetry');
    await page.unroute('**/health');
    await state('events-kpi','success');
    await page.waitForFunction(() => document.querySelector('[data-testid="system-status"]')?.textContent.includes('ONLINE'),null,{timeout:25000});
    passed('Polling recovers automatically');
    for (const [width,height] of [[1440,900],[1366,768]]) {
      await page.setViewportSize({width,height});
      const fits=await page.evaluate(()=>{const r=document.querySelector('.fit-shell').getBoundingClientRect();return r.right<=innerWidth+1&&r.bottom<=innerHeight+1;});
      assert.ok(fits);
      await page.screenshot({path:path.join(output,`${width}x${height}.png`)});
    }
    passed('Approved canvas fits 1440×900 and 1366×768');
    await page.setViewportSize({width:1920,height:1080});
    await page.goto(origin+'/#telemetry');
    await page.locator('[data-testid="events-donut"][data-state="success"]').waitFor();
    assert.match(await page.locator('[data-testid="events-donut"]').innerText(),/STALE_DISCONNECT/);
    assert.doesNotMatch(await page.locator('[data-testid="events-donut"]').innerText(),/OTHER/);
    passed('Telemetry route shares real charts and exact event categories');
    // Delayed responses exercise timeout and ensure sequential polling.
    let inFlight=0,maxInFlight=0,requests=0;
    await page.route('**/telemetry',async route=>{
      inFlight++;requests++;maxInFlight=Math.max(maxInFlight,inFlight);
      await sleep(9000);
      try {await route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(after)});}catch{}
      inFlight--;
    });
    await page.waitForFunction(()=>document.querySelector('[data-testid="events-donut"]')?.getAttribute('data-state')==='stale',null,{timeout:25000});
    assert.equal(maxInFlight,1);
    passed('Slow telemetry times out without overlapping polling');
    const requestCount=requests;
    await page.goto('about:blank');
    await sleep(6500);
    assert.equal(requests,requestCount);
    passed('No further polling after page teardown');
    assert.deepEqual(errors,[]);
    passed('No uncaught browser errors');
    fs.writeFileSync(path.join(output,'test-results.json'),JSON.stringify({passed:results},null,2));
  } finally { await browser.close(); }
})().catch(error=>{console.error(error);process.exitCode=1;});
