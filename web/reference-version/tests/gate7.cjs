const {chromium}=require('C:/Users/joaob/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const api='http://127.0.0.1:18098',origin='http://localhost:5198',out=path.resolve(__dirname,'../evidence/gate7');fs.mkdirSync(out,{recursive:true});
const results=[],errors=[],requests=[];const pass=s=>{results.push(s);console.log('PASS',s)};
const get=async p=>{const r=await fetch(api+p);assert.equal(r.status,200);return r.json()};
const post=async(p,b={})=>{const r=await fetch(api+p,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)});assert.ok(r.ok,await r.clone().text());return r.json()};
const wait=async(fn)=>{for(let i=0;i<100;i++){if(await fn())return;await new Promise(r=>setTimeout(r,200));}throw Error('condition timed out')};
(async()=>{
 assert.deepEqual(await get('/api/v1/events/recent'),[]);pass('Fresh API execution starts with empty feed');
 const stamp=String(Date.now()).slice(-10),sub=await post('/api/v1/subscribers',{imsi:'72496'+stamp,msisdn:'+5516'+stamp});await post('/api/v1/subscribers/'+sub.id+'/activate');
 const device=await post('/api/v1/devices',{subscriber_id:sub.id,imei:'86000'+stamp,technology:'5G'});
 const s=await post('/api/v1/sessions/attach',{device_id:device.id,cell_id:'CELL-SP-001'});
 const h=await post('/api/v1/sessions/'+s.id+'/handover',{target_cell_id:'CELL-SP-002'});assert.equal(h.id,s.id);assert.equal(h.ip_address,s.ip_address);
 await post('/api/v1/sessions/'+s.id+'/detach');
 await wait(async()=> (await get('/api/v1/events/recent')).length===3);
 const hero=await get('/api/v1/events/recent');assert.deepEqual(hero.map(e=>e.type),['DETACH','CELL_HANDOVER','ATTACH']);assert.equal(hero[1].from_cell_id,'CELL-SP-001');assert.equal(hero[1].to_cell_id,'CELL-SP-002');assert.ok(hero.every(e=>e.device_id===device.id&&e.subscriber_id===sub.id&&e.session_id===s.id));assert.equal(hero[0].disconnect_reason,'VOLUNTARY_DETACH');pass('Real Attach → Handover → Detach; UUID/IP and original cell preserved');
 const active=await post('/api/v1/sessions/attach',{device_id:device.id,cell_id:'CELL-SP-001'});
 const replacement=await post('/api/v1/sessions/attach',{device_id:device.id,cell_id:'CELL-SP-002'});
 await wait(async()=> (await get('/api/v1/events/recent')).length===6);
 const feed=await get('/api/v1/events/recent');assert.equal(feed[0].type,'ATTACH');assert.equal(feed[0].session_id,replacement.id);assert.equal(feed[1].type,'STALE_DISCONNECT');assert.equal(feed[1].session_id,active.id);assert.equal(feed[1].disconnect_reason,'STALE_DISCONNECT');assert.equal(feed[1].cell_id,'CELL-SP-001');pass('Real re-attach emits STALE_DISCONNECT for old session then ATTACH for replacement');
 const telemetry=await get('/telemetry');assert.deepEqual(telemetry.events_total,{attach:3,cell_handover:1,detach:1,stale_disconnect:1});pass('Telemetry counters count each event once');
 if(process.env.GATE7_RESTART_ONLY){fs.writeFileSync(path.join(out,'restart.json'),JSON.stringify({results,feed,telemetry},null,2));return;}
 const browser=await chromium.launch({channel:'chrome',headless:true}),page=await browser.newPage({viewport:{width:1920,height:1080}});
 page.on('pageerror',e=>errors.push(e.message));page.on('request',r=>{if(r.url().includes('/events/recent'))requests.push({time:Date.now(),url:r.url()})});
 try{
  await page.goto(origin);await page.locator('[data-testid="recent-events"][data-state="success"]').waitFor();assert.equal(await page.getByTestId('recent-event-row').count(),5);assert.ok((await page.getByTestId('recent-events').innerText()).includes('STALE_DISCONNECT'));assert.equal(await page.locator('[data-testid="events-kpi"] strong').innerText(),'6');
  for(const [width,height] of [[1920,1080],[1440,900],[1366,768]]){await page.setViewportSize({width,height});await page.waitForTimeout(200);await page.screenshot({path:path.join(out,`overview-${width}x${height}.png`)});}pass('Overview real feed at three resolutions; Total Events remains telemetry');
  await page.goto(origin+'/#events');await page.getByTestId('recent-event-row').nth(5).waitFor();await page.getByTestId('recent-event-row').first().getByRole('button').click();assert.ok((await page.getByRole('dialog').innerText()).includes(device.id));await page.getByRole('button',{name:'Close event details'}).click();pass('Events uses shared snapshot; full UUID details accessible');
  const pattern='**/api/v1/events/recent';await page.route(pattern,r=>r.fulfill({status:503,body:'unavailable'}));await page.locator('[data-testid="recent-events"][data-state="stale"]').waitFor({timeout:15000});assert.equal(await page.getByTestId('recent-event-row').count(),6);pass('Failed read preserves last six events as stale');
  await page.reload();await page.locator('[data-testid="recent-events"][data-state="error"]').waitFor();assert.equal(await page.getByTestId('recent-event-row').count(),0);assert.ok(!(await page.getByTestId('recent-events').innerText()).includes('No events observed'));await page.unrouteAll({behavior:'wait'});await page.locator('[data-testid="recent-events"][data-state="success"]').waitFor({timeout:15000});pass('Initial failure is error, not false empty success; polling recovers');
  let activeRequests=0,max=0,starts=[],ends=[];
  await page.route(pattern,async r=>{starts.push(Date.now());activeRequests++;max=Math.max(max,activeRequests);try{const response=await r.fetch();await new Promise(resolve=>setTimeout(resolve,5500));await r.fulfill({response});}finally{activeRequests--;ends.push(Date.now())}});
  await wait(()=>starts.length>=1);await page.waitForTimeout(7000);assert.equal(starts.length,1);await wait(()=>starts.length>=2);assert.ok(starts[1]-ends[0]>=4800);assert.equal(max,1);await page.unrouteAll({behavior:'wait'});pass('Slow feed polling never overlaps; next round waits five seconds after completion');
  await page.goto('about:blank');const count=requests.length;await page.waitForTimeout(6000);assert.equal(requests.length,count);assert.deepEqual(errors,[]);pass('Unmount stops feed polling; zero browser errors');
 }finally{fs.writeFileSync(path.join(out,'results.json'),JSON.stringify({results,hero,feed,telemetry,requests,errors},null,2));await browser.close();}
})().catch(e=>{console.error(e);process.exitCode=1});

