const {chromium}=require('C:/Users/joaob/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const api=process.env.GATE6_API||'http://127.0.0.1:18089',origin=process.env.GATE6_ORIGIN||'http://127.0.0.1:5189',out=path.resolve(__dirname,'../evidence/gate6');fs.mkdirSync(out,{recursive:true});
const results=[],requests=[],errors=[],hero=[];const pass=s=>{results.push(s);console.log('PASS',s)};
const write=async(p,body={})=>{const r=await fetch(api+p,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});assert.ok(r.ok,await r.clone().text());return r.json()};
(async()=>{
 const stamp=String(Date.now()).slice(-10),devices=[];
 const sub=await write('/api/v1/subscribers',{imsi:'72496'+stamp,msisdn:'+5516'+stamp});await write('/api/v1/subscribers/'+sub.id+'/activate');
 const create=async()=>{const d=await write('/api/v1/devices',{subscriber_id:sub.id,imei:'86'+String(devices.length).padStart(3,'0')+stamp,technology:'5G'});devices.push(d);return d};
 const first=await create();
 const browser=await chromium.launch({channel:'chrome',headless:true});
 const map=await browser.newPage({viewport:{width:1920,height:1080}}),operations=await browser.newPage({viewport:{width:1920,height:1080}});
 map.on('request',r=>requests.push({method:r.method(),url:r.url(),at:Date.now()}));map.on('pageerror',e=>errors.push(e.message));operations.on('pageerror',e=>errors.push(e.message));
 const state=async s=>map.waitForFunction(s=>document.querySelector('[data-testid="topology-state"]')?.dataset.state===s,s);
 const connected=async n=>map.waitForFunction(n=>document.querySelectorAll('[data-testid="topology-device"][data-state="connected"]').length===n,n);
 const shots=async label=>{for(const [width,height] of [[1920,1080],[1440,900],[1366,768]]){await map.setViewportSize({width,height});await map.screenshot({path:path.join(out,`${label}-${width}x${height}.png`)});}};
 const action=async name=>{const response=operations.waitForResponse(r=>r.url().includes('/api/v1/sessions/')&&r.request().method()==='POST');await operations.getByRole('button',{name,exact:true}).click();const r=await response;assert.ok(r.ok());const data=await r.json();hero.push({action:name,...data});return data};
 try{
   await map.goto(origin+'/#overview');await state('success');await connected(0);assert.match(await map.getByTestId('topology-state').innerText(),/1 without session/);await shots('no-sessions');pass('One real Device without session has no green link');
   await operations.goto(origin+'/#sessions');await operations.getByLabel('Session device').selectOption(first.id);await operations.getByTestId('session-empty').waitFor();await operations.getByLabel('Session cell').selectOption('CELL-SP-001');
   const attached=await action('Attach');await connected(1);assert.equal(await map.getByTestId('topology-link').getAttribute('data-cell-id'),'CELL-SP-001');assert.equal(await map.getByTestId('topology-device').getAttribute('data-session-id'),attached.id);await shots('connected');pass('Attach in Sessions becomes a real topology connection on next poll');
   await operations.getByLabel('Session cell').selectOption('CELL-SP-002');const moved=await action('Handover');assert.equal(moved.id,attached.id);assert.equal(moved.ip_address,attached.ip_address);
   await map.waitForFunction(()=>document.querySelector('[data-testid="topology-link"]')?.getAttribute('data-cell-id')==='CELL-SP-002');assert.equal(await map.getByTestId('topology-device').getAttribute('data-ip'),attached.ip_address);assert.equal(await map.locator('.handover-path').count(),0);await shots('handover');pass('Handover changes association; same UUID/IP, no permanent physical trajectory');
   await action('Detach');await connected(0);await state('success');assert.match(await map.getByTestId('topology-state').innerText(),/1 without session/);assert.equal((await(await fetch(api+'/api/v1/devices')).json()).length,1);await shots('detached');pass('Detach removes connection without deleting registered Device');
   await write('/api/v1/sessions/attach',{device_id:first.id,cell_id:'CELL-SP-001'});
   for(let i=1;i<10;i++){const d=await create();await write('/api/v1/sessions/attach',{device_id:d.id,cell_id:'CELL-SP-001'});if(i===4){await map.reload();await connected(5);await map.screenshot({path:path.join(out,'five-devices.png')});pass('Five real Devices rendered');}}
   await map.reload();await connected(10);await map.screenshot({path:path.join(out,'ten-devices.png')});assert.equal(await map.getByTestId('live-session-row').count(),5);
   const transforms=await map.getByTestId('topology-device').evaluateAll(nodes=>nodes.map(n=>n.getAttribute('transform')));assert.equal(new Set(transforms).size,10);pass('Ten Devices on one cell have distinct deterministic positions; preview shares snapshot');
   await map.getByTestId('topology-device').first().click();assert.match(await map.locator('.topology-tooltip').innerText(),/Device technology: 5G/);assert.match(await map.locator('.topology-tooltip').innerText(),/LTE/);await map.getByTestId('topology-device').first().click();pass('Device 5G on LTE cell displayed honestly without compatibility filtering');
   const pattern='**/api/v1/sessions?device_id=*';let mode='partial';
   await map.route(pattern,async route=>{const id=new URL(route.request().url()).searchParams.get('device_id');if(mode==='all'||id===first.id)await route.fulfill({status:404,contentType:'application/json',body:JSON.stringify({code:'ROUTE_NOT_FOUND',message:'Injected read error'})});else await route.continue()});
   await state('partial');assert.equal(await map.locator('[data-testid="topology-device"][data-state="stale"]').count(),1);assert.equal(await map.locator('[data-testid="topology-device"][data-state="connected"]').count(),9);await map.screenshot({path:path.join(out,'partial.png')});pass('Partial failure retains stale Device, never disconnected');
   mode='all';await state('stale');await map.reload();await state('error');assert.match(await map.getByTestId('topology-state').innerText(),/10 unknown/);assert.equal(await map.getByTestId('topology-link').count(),0);pass('Total failure: stale with known snapshot, error/unknown without snapshot');
   await map.unroute(pattern);await state('success');await connected(10);pass('Automatic next-round recovery');
   // Instrument a slow round (> polling interval) without changing real responses.
   await map.goto(origin+'/#devices');await map.waitForTimeout(500);
   let active=0,max=0,started=0,finished=0,lastEnd=0,firstNext=0;const ids=[],routeFailures=[];
   await map.route(pattern,async route=>{const ordinal=++started;ids.push(new URL(route.request().url()).searchParams.get('device_id'));if(ordinal===11)firstNext=Date.now();active++;max=Math.max(max,active);try{const response=await route.fetch();await new Promise(r=>setTimeout(r,ordinal===1?5500:100));await route.fulfill({response});}catch(error){routeFailures.push(error.message)}finally{active--;finished++;if(finished===10)lastEnd=Date.now();}});
   await map.goto(origin+'/#overview');await map.waitForTimeout(7500);assert.equal(started,10);assert.equal(finished,10);assert.equal(new Set(ids).size,10);assert.ok(max<=4);await map.waitForTimeout(5000);assert.ok(started>=11);assert.ok(firstNext-lastEnd>=4800);await map.unrouteAll({behavior:'wait'});assert.deepEqual(routeFailures,[]);pass(`Concurrency capped at ${max}; slow round not overlapped; >=5s after completion`);
   await map.goto(origin+'/#devices');const before=requests.filter(r=>r.url.includes('/api/v1/sessions')).length;await map.waitForTimeout(6500);assert.equal(requests.filter(r=>r.url.includes('/api/v1/sessions')).length,before);pass('Topology polling cancelled when leaving topology routes');
   assert.ok(requests.filter(r=>r.url.includes('/api/v1/sessions')).every(r=>new URL(r.url).searchParams.has('device_id')));assert.deepEqual(errors,[]);pass('Existing per-Device contract only; zero uncaught browser errors');
 }finally{fs.writeFileSync(path.join(out,'results.json'),JSON.stringify({results,hero,requests,errors},null,2));await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
