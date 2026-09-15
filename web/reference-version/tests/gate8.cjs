const {chromium}=require('C:/Users/joaob/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const api=process.env.GATE8_API||'http://127.0.0.1:18108',origin=process.env.GATE8_ORIGIN||'http://localhost:5208',out=path.resolve(__dirname,'../evidence/gate8');fs.mkdirSync(out,{recursive:true});
const results=[],snapshots=[],errors=[],requests=[];const pass=s=>{results.push(s);console.log('PASS',s)};
const post=async(p,body={})=>{const r=await fetch(api+p,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});assert.ok(r.ok,await r.clone().text());return r.json()};
const get=async p=>{const r=await fetch(api+p);assert.equal(r.status,200);return r.json()};
const wait=async f=>{for(let i=0;i<120;i++){if(await f())return;await new Promise(r=>setTimeout(r,200));}throw Error('condition timed out')};
(async()=>{
 const initial=await get('/api/v1/network/ip-pool');assert.deepEqual(initial,{cidr:'10.45.0.0/16',capacity:65533,allocated:0,available:65533,utilization_percent:0});snapshots.push({step:'initial',...initial});
 const browser=await chromium.launch({channel:'chrome',headless:true}),page=await browser.newPage({viewport:{width:1920,height:1080}}),pattern='**/api/v1/network/ip-pool';
 page.on('pageerror',e=>errors.push(e.message));page.on('request',r=>{if(r.url().includes('/network/ip-pool'))requests.push({url:r.url(),time:Date.now(),method:r.method()})});
 const state=async s=>page.locator(`[data-testid="ip-pool"][data-state="${s}"]`).waitFor({timeout:20000});
 const value=async n=>{await state('success');await wait(async()=>await page.getByTestId('pool-allocated').innerText()===String(n));const p=await get('/api/v1/network/ip-pool');assert.equal(p.allocated,n);assert.equal(p.available,65533-n);assert.equal(p.utilization_percent,n/65533*100);snapshots.push(p)};
 const shots=async name=>{for(const[width,height]of[[1920,1080],[1440,900],[1366,768]]){await page.setViewportSize({width,height});await page.waitForTimeout(150);const card=await page.getByTestId('ip-pool').boundingBox(),note=await page.getByTestId('ip-pool').getByRole('status').boundingBox();assert.ok(note.y+note.height<=card.y+card.height,'card content clipped');await page.screenshot({path:path.join(out,`${name}-${width}x${height}.png`)})}};
 try{
  await page.route(pattern,async r=>{await new Promise(resolve=>setTimeout(resolve,1500));try{await r.continue()}catch{/* StrictMode can cancel its first mount request. */}});
  await page.goto(origin);await state('loading');assert.equal(await page.getByTestId('pool-allocated').innerText(),'—');await value(0);await page.unrouteAll({behavior:'wait'});await shots('empty');pass('Loading is unknown, then authoritative empty pool at three resolutions');
  const stamp=String(Date.now()).slice(-10),sub=await post('/api/v1/subscribers',{imsi:'72496'+stamp,msisdn:'+5516'+stamp});await post('/api/v1/subscribers/'+sub.id+'/activate');
  const create=i=>post('/api/v1/devices',{subscriber_id:sub.id,imei:'8600'+i+stamp,technology:'5G'});
  const a=await create(1),b=await create(2);
  const sa=await post('/api/v1/sessions/attach',{device_id:a.id,cell_id:'CELL-SP-001'});await value(1);assert.equal(sa.ip_address,'10.45.0.2');pass('First Attach changes pool from zero to one');
  const sb=await post('/api/v1/sessions/attach',{device_id:b.id,cell_id:'CELL-SP-001'});await value(2);assert.equal(sb.ip_address,'10.45.0.3');await page.reload();await value(2);await shots('allocated');pass('Second Attach: two allocated, distinct IPs from the same allocator');
  const handover=await post('/api/v1/sessions/'+sb.id+'/handover',{target_cell_id:'CELL-SP-002'});await value(2);assert.equal(handover.id,sb.id);assert.equal(handover.ip_address,sb.ip_address);pass('Handover preserves allocation count and Session IP');
  await post('/api/v1/sessions/'+sa.id+'/detach');await value(1);await shots('detached');pass('Detach reduces allocated to one; available increases');
  const replacement=await post('/api/v1/sessions/attach',{device_id:b.id,cell_id:'CELL-SP-002'});await value(1);assert.notEqual(replacement.id,sb.id);assert.equal(replacement.ip_address,sa.ip_address);const feed=await get('/api/v1/events/recent');assert.ok(feed.some(e=>e.type==='STALE_DISCONNECT'&&e.session_id===sb.id));pass('Re-attach ends with one allocated; released address reused; Recent Events preserved');
  await page.route(pattern,r=>r.fulfill({status:503,body:'Injected pool read failure'}));await state('stale');assert.equal(await page.getByTestId('pool-allocated').innerText(),'1');pass('Failed read keeps last allocated value as stale');
  await page.reload();await state('error');assert.equal(await page.getByTestId('pool-allocated').innerText(),'—');await page.unrouteAll({behavior:'wait'});await value(1);pass('Initial error is unknown, not zero; polling recovers');
  await page.route(pattern,r=>r.fulfill({status:200,contentType:'application/json',body:JSON.stringify({...initial,allocated:1})}));await state('stale');assert.equal(await page.getByTestId('pool-allocated').innerText(),'1');await page.unrouteAll({behavior:'wait'});await value(1);pass('Inconsistent JSON rejected without corrupting last snapshot');
  let active=0,max=0,starts=[],ends=[];await page.route(pattern,async r=>{starts.push(Date.now());active++;max=Math.max(max,active);try{const response=await r.fetch();await new Promise(resolve=>setTimeout(resolve,5500));await r.fulfill({response});}finally{active--;ends.push(Date.now())}});await wait(()=>starts.length>=1);await page.waitForTimeout(7000);assert.equal(starts.length,1);await wait(()=>starts.length>=2);assert.equal(max,1);assert.ok(starts[1]-ends[0]>=4800);await page.unrouteAll({behavior:'wait'});pass('Slow reads never overlap; next poll five seconds after completion');
  await page.goto(origin+'/#devices');const before=requests.length;await page.waitForTimeout(6000);assert.equal(requests.length,before);assert.deepEqual(errors,[]);assert.ok(requests.every(r=>r.method==='GET'));pass('Unmount cancels pool polling; read-only transport; zero uncaught JavaScript errors');
 }finally{fs.writeFileSync(path.join(out,'results.json'),JSON.stringify({results,snapshots,requests,errors},null,2));await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
