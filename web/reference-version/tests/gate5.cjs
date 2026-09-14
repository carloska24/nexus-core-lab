const {chromium}=require('C:/Users/joaob/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const api='http://127.0.0.1:18088',origin='http://127.0.0.1:5188';
const out=path.resolve(__dirname,'../evidence/gate5');fs.mkdirSync(out,{recursive:true});
const results=[],requests=[],errors=[];const pass=x=>{results.push(x);console.log('PASS',x)};
const post=async(p,body={})=>{const r=await fetch(api+p,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});assert.ok(r.ok,await r.clone().text());return r.json()};
(async()=>{
 const suffix=String(Date.now()).slice(-10);
 const sub=await post('/api/v1/subscribers',{imsi:'72495'+suffix,msisdn:'+5515'+suffix});await post('/api/v1/subscribers/'+sub.id+'/activate');
 const a=await post('/api/v1/devices',{subscriber_id:sub.id,imei:'86001'+suffix,technology:'5G'});
 const b=await post('/api/v1/devices',{subscriber_id:sub.id,imei:'86002'+suffix,technology:'LTE'});
 const browser=await chromium.launch({channel:'chrome',headless:true}),page=await browser.newPage({viewport:{width:1920,height:1080}});
 page.on('pageerror',e=>errors.push(e.message));page.on('request',r=>requests.push({method:r.method(),url:r.url()}));
 const ready=async s=>page.waitForFunction(s=>document.querySelector('[data-testid="session-panel"]')?.dataset.state===s,s);
 const status=async s=>page.waitForFunction(s=>document.querySelector('[data-testid="session-status"]')?.textContent===s,s);
 const action=async name=>{const response=page.waitForResponse(r=>r.url().includes('/api/v1/sessions/')&&r.request().method()==='POST');await page.getByRole('button',{name,exact:true}).click();const r=await response;assert.ok(r.ok());await ready('success');return r.json()};
 try{
   await page.goto(origin+'/#sessions');await page.getByLabel('Session device').selectOption(a.id);await ready('success');await page.getByTestId('session-empty').waitFor();pass('Active-session 404 with correct code becomes confirmed absence');
   const first=await action('Attach');await status('CONNECTED');assert.equal(first.device_id,a.id);assert.equal(first.cell_id,'CELL-SP-001');assert.ok(first.ip_address);pass('Real Attach returns CONNECTED with allocated IP');
   await page.getByLabel('Session cell').selectOption('CELL-SP-002');const moved=await action('Handover');assert.equal(moved.id,first.id);assert.equal(moved.ip_address,first.ip_address);assert.equal(moved.cell_id,'CELL-SP-002');pass('Real Handover preserves session UUID and IP');
   for(const [width,height] of [[1920,1080],[1440,900],[1366,768]]){await page.setViewportSize({width,height});await page.screenshot({path:path.join(out,`connected-${width}x${height}.png`)});const rect=await page.getByRole('button',{name:'Detach',exact:true}).boundingBox();assert.ok(rect.y+rect.height<=height,'Detach must fit viewport');}
   const dropped=await action('Detach');await status('DISCONNECTED');assert.equal(dropped.disconnect_reason,'VOLUNTARY_DETACH');assert.ok(dropped.closed_at);await page.screenshot({path:path.join(out,'disconnected-1366x768.png')});pass('Real Detach exposes closure timestamp and reason');
   await page.getByRole('button',{name:'Refresh session'}).click();await ready('success');await page.getByTestId('session-empty').waitFor();
   await page.getByLabel('Session device').selectOption(b.id);await ready('success');await page.getByTestId('session-empty').waitFor();assert.equal(await page.getByTestId('session-details').count(),0);pass('Device switch clears previous session, no cross-device snapshot');
   await action('Attach');
   const getPattern='**/api/v1/sessions?device_id=*';await page.route(getPattern,r=>r.abort('connectionrefused'));
   await page.getByRole('button',{name:'Refresh session'}).click();await ready('stale');await status('CONNECTED');assert.equal(await page.getByRole('button',{name:'Detach',exact:true}).isDisabled(),true);
   await page.getByLabel('Session device').selectOption(a.id);await ready('error');assert.equal(await page.getByTestId('session-empty').count(),0);assert.equal(await page.getByRole('button',{name:'Attach',exact:true}).isDisabled(),true);
   await page.unroute(getPattern);await page.getByRole('button',{name:'Refresh session'}).click();await ready('success');await page.getByTestId('session-empty').waitFor();pass('Offline stale preserves snapshot; initial error is not no-session; refresh recovers');
   await page.route(getPattern,r=>r.fulfill({status:404,contentType:'application/json',body:JSON.stringify({code:'ROUTE_NOT_FOUND',message:'Unknown route'})}));await page.getByRole('button',{name:'Refresh session'}).click();await ready('stale');assert.match(await page.getByRole('alert').innerText(),/ROUTE_NOT_FOUND/);await page.unroute(getPattern);await page.getByRole('button',{name:'Refresh session'}).click();await ready('success');pass('Unrelated 404 remains an error');
   await post('/api/v1/subscribers/'+sub.id+'/suspend',{reason:'Gate 5 eligibility test'});
   // Backend checker decides whether a suspended owner affects attach; inspect its actual response.
   const req=page.waitForResponse(r=>r.url().endsWith('/api/v1/sessions/attach')&&r.request().method()==='POST');await page.getByRole('button',{name:'Attach',exact:true}).click();const actual=await req;
   if(actual.status()===422){await ready('stale');assert.match(await page.getByRole('alert').innerText(),/422/);pass('Backend rejects ineligible Device with visible 422');}
   else {assert.ok(actual.ok());await status('CONNECTED');pass('Backend allows registered Device despite suspended owner; frontend honors backend contract');await action('Detach');}
   await post('/api/v1/subscribers/'+sub.id+'/activate');await page.getByRole('button',{name:'Refresh session'}).click();await ready('success');
   await page.route('**/api/v1/sessions/attach',r=>r.fulfill({status:500,contentType:'application/json',body:JSON.stringify({code:'INTERNAL_SERVER_ERROR',message:'Injected write failure'})}));await page.getByRole('button',{name:'Attach',exact:true}).click();await ready('stale');assert.match(await page.getByRole('alert').innerText(),/500/);assert.equal(await page.getByTestId('session-status').count(),0);await page.unroute('**/api/v1/sessions/attach');await page.getByRole('button',{name:'Refresh session'}).click();await ready('success');pass('Failed write has no optimistic CONNECTED and requires refresh');
   const begin=requests.length;await page.waitForTimeout(11000);assert.equal(requests.slice(begin).filter(r=>r.url.includes('/api/v1/sessions')).length,0);assert.ok(requests.slice(begin).some(r=>r.url.includes('/telemetry')));pass('No Sessions polling; existing telemetry continues');
   assert.ok(requests.filter(r=>r.method==='GET'&&r.url.includes('/api/v1/sessions')).every(r=>new URL(r.url).searchParams.has('device_id')));assert.deepEqual(errors,[]);pass('No global Sessions query; zero uncaught JS errors');
 }finally{fs.writeFileSync(path.join(out,'results.json'),JSON.stringify({results,requests,errors},null,2));await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
