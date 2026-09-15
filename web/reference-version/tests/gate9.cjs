const {chromium}=require('C:/Users/joaob/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const origin=process.env.GATE9_ORIGIN||'http://localhost:5209',api=process.env.GATE9_API||'http://127.0.0.1:18109';
const out=path.resolve(__dirname,'../evidence/gate9');fs.mkdirSync(out,{recursive:true});
const results=[],errors=[];const pass=s=>{results.push(s);console.log('PASS',s)};
const memory={mode:'MEMORY',database_configured:false,database_status:'NOT_APPLICABLE'};
(async()=>{
 const health=await (await fetch(api+'/health')).json(),storage=await(await fetch(api+'/api/v1/system/storage')).json();assert.equal(health.status,'ok');assert.deepEqual(storage,memory);
 const browser=await chromium.launch({channel:'chrome',headless:true}),page=await browser.newPage({viewport:{width:1920,height:1080}}),pattern='**/api/v1/system/storage';
 page.on('pageerror',e=>errors.push(e.message));
 const state=s=>page.locator(`[data-testid="storage-status"][data-state="${s}"]`).waitFor({timeout:45000});
 const label=()=>page.getByTestId('storage-status').innerText();
 try {
  await page.route(pattern,async r=>{await new Promise(done=>setTimeout(done,1200));try{await r.continue()}catch{}});
  await page.goto(origin);await state('loading');assert.match(await label(),/Checking/);await state('success');assert.match(await label(),/MEMORY/);assert.doesNotMatch(await label(),/OFFLINE/);await page.unrouteAll({behavior:'wait'});pass('Real MEMORY and health OK; loading then success');
  for(const[width,height]of[[1920,1080],[1440,900],[1366,768]]){await page.setViewportSize({width,height});await page.waitForTimeout(200);const box=await page.getByTestId('storage-status').boundingBox();assert.ok(box.x>=0&&box.x+box.width<=width&&box.y+box.height<=height);await page.screenshot({path:path.join(out,`memory-${width}x${height}.png`)});}pass('Real MEMORY screenshots at three resolutions');
  await page.goto(origin+'/#database');await page.getByTestId('workspace-status').waitFor();assert.match(await page.getByTestId('workspace-status').innerText(),/MEMORY/);assert.match(await page.locator('.workspace-card').innerText(),/NOT_APPLICABLE/);pass('Database page reflects authoritative MEMORY');
  for(const status of ['AVAILABLE','UNAVAILABLE']){
   await page.route(pattern,r=>r.fulfill({status:200,contentType:'application/json',body:JSON.stringify({mode:'POSTGRESQL',database_configured:true,database_status:status})}));await page.reload();await state('success');assert.match(await label(),new RegExp(status==='AVAILABLE'?'POSTGRESQL · ONLINE':'POSTGRESQL · OFFLINE'));assert.match(await page.getByTestId('system-status').innerText(),/ONLINE/);await page.unrouteAll({behavior:'wait'});pass('Controlled PostgreSQL '+status+' independent of API liveness');
  }
  await page.route(pattern,r=>r.fulfill({status:503,body:'injected read failure'}));await state('stale');assert.match(await label(),/Stale/);assert.equal(await page.locator('.infra > i').last().getAttribute('data-status'),'unknown');pass('Request failure preserves last snapshot as stale, never a fresh OFFLINE');
  await page.reload();await state('error');assert.match(await label(),/Unknown/);assert.doesNotMatch(await label(),/OFFLINE/);await page.unrouteAll({behavior:'wait'});await state('success');assert.match(await label(),/MEMORY/);pass('Initial error unknown and automatic recovery');
  await page.route(pattern,r=>r.fulfill({status:200,contentType:'application/json',body:JSON.stringify({...memory,database_status:'UNAVAILABLE'})}));await state('stale');assert.match(await label(),/MEMORY/);await page.unrouteAll({behavior:'wait'});pass('Inconsistent storage response rejected');
  let starts=[],ends=[],active=0,max=0;await page.route(pattern,async r=>{starts.push(Date.now());active++;max=Math.max(max,active);try{await new Promise(done=>setTimeout(done,1500));await r.fulfill({status:200,contentType:'application/json',body:JSON.stringify(memory)})}finally{active--;ends.push(Date.now())}});
  const deadline=Date.now()+35000;while(starts.length<2&&Date.now()<deadline)await page.waitForTimeout(200);assert.ok(starts.length>=2);assert.equal(max,1);assert.ok(starts[1]-ends[0]>=9800);await page.unrouteAll({behavior:'wait'});pass('Shared polling waits ten seconds after completion without overlap');
  await page.reload();await state('success');assert.match(await label(),/MEMORY/);
  const slow=require('node:http').createServer((req,res)=>{const timer=setTimeout(()=>{res.setHeader('Access-Control-Allow-Origin','*');res.setHeader('Content-Type','application/json');res.end(JSON.stringify(memory))},12000);res.on('close',()=>clearTimeout(timer))});
  await new Promise(done=>slow.listen(0,'127.0.0.1',done));
  try{await page.route(pattern,r=>r.continue({url:`http://127.0.0.1:${slow.address().port}/api/v1/system/storage`}));await state('stale');assert.match(await label(),/Stale/);pass('Actual delayed HTTP response reaches transport timeout and preserves snapshot')}finally{await page.unrouteAll({behavior:'wait'});slow.closeAllConnections();await new Promise(done=>slow.close(done))}

  let count=0;page.on('request',r=>{if(r.url().includes('/system/storage'))count++});await page.goto('about:blank');const before=count;await page.waitForTimeout(11000);assert.equal(count,before);assert.deepEqual(errors,[]);pass('Unmount stops polling; zero uncaught JavaScript errors');
 } finally {fs.writeFileSync(path.join(out,'results.json'),JSON.stringify({real:{health,storage},controlledPostgres:true,results,errors},null,2));await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
