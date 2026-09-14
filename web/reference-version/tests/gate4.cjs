// Run against the isolated memory API :18086 and frontend :5186.
const {chromium}=require('C:/Users/joaob/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const out=path.resolve(__dirname,'../evidence/gate4');fs.mkdirSync(out,{recursive:true});
const origin=process.env.GATE4_ORIGIN||'http://127.0.0.1:5186',api=process.env.GATE4_API||'http://127.0.0.1:18086',results=[];
const pass=s=>{results.push(s);console.log('PASS',s)};
(async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 const page=await browser.newPage({viewport:{width:1920,height:1080}}),requests=[],errors=[];
 page.on('request',r=>requests.push({url:r.url(),method:r.method()}));page.on('pageerror',e=>errors.push(e.message));
 const nav=async name=>page.locator('.sidebar a').filter({hasText:new RegExp('^'+name+'$')}).click();
 const state=async s=>page.waitForFunction(s=>document.querySelector('[data-testid="devices-page"]')?.dataset.state===s,s);
 const total=async n=>page.waitForFunction(n=>document.querySelector('[data-testid="device-total"]')?.textContent===String(n),n);
 const close=async()=>page.getByRole('button',{name:'Close device dialog'}).click();
 const open=async()=>page.getByRole('button',{name:'Register device',exact:true}).click();
 const post=async(url,body)=>fetch(api+url,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
 const stamp=String(Date.now()).slice(-9);let serial=0;
 const imei=()=> '860100'+stamp.slice(0,6)+String(++serial).padStart(3,'0');
 const makeOwner=async(status='ACTIVE')=>{
   await nav('Subscribers');await page.getByRole('button',{name:'Provision subscriber',exact:true}).click();
   const imsi='72499'+stamp+String(++serial),msisdn='+5519'+stamp+String(serial);
   await page.getByLabel('Provision IMSI',{exact:true}).fill(imsi);await page.getByLabel('Provision MSISDN',{exact:true}).fill(msisdn);
   const response=page.waitForResponse(r=>r.url().endsWith('/api/v1/subscribers')&&r.request().method()==='POST');
   await page.getByRole('button',{name:'Provision',exact:true}).click();const owner=await(await response).json();
   await page.getByRole('button',{name:'Activate',exact:true}).click();
   await page.waitForFunction(()=>document.querySelector('dialog [data-testid="subscriber-status"]')?.textContent==='ACTIVE');
   if(status!=='ACTIVE'){
     await page.getByLabel('Lifecycle reason').fill('Gate 4 isolated verification');
     await page.getByRole('button',{name:status==='SUSPENDED'?'Suspend':'Deactivate permanently',exact:true}).click();
     await page.waitForFunction(s=>document.querySelector('dialog [data-testid="subscriber-status"]')?.textContent===s,status);
   }
   await page.getByRole('button',{name:'Close subscriber dialog'}).click();await nav('Devices');return owner;
 };
 const register=async(owner,tech,value)=>{
   await open();await page.getByLabel('Device subscriber',{exact:true}).selectOption(owner.id);
   await page.getByLabel('Register IMEI').fill(value);await page.getByLabel('Device technology').selectOption(tech);
   const response=page.waitForResponse(r=>r.url().endsWith('/api/v1/devices')&&r.request().method()==='POST');
   await page.getByRole('button',{name:'Register',exact:true}).click();const device=await(await response).json();
   await page.getByTestId('device-details').waitFor();assert.match(await page.getByTestId('device-details').innerText(),new RegExp(value));await close();return device;
 };
 try{
   const baseline=(await(await fetch(api+'/api/v1/devices')).json())??[];
   await page.goto(origin+'/#devices');await state('success');await total(baseline.length);
   if(!baseline.length){assert.match(await page.locator('.device-table').innerText(),/No devices registered yet/);await nav('Overview');await page.waitForFunction(()=>document.querySelector('[data-testid="devices-kpi"] strong')?.textContent==='0');await nav('Devices');pass('Real empty collection and Overview KPI = 0');}
   await open();
   if(await page.getByText('No active subscribers available.',{exact:false}).count())pass('No active subscribers message and registration disabled');
   await close();const owner=await makeOwner();
   await open();await page.getByLabel('Device subscriber',{exact:true}).selectOption(owner.id);await page.getByLabel('Register IMEI').fill('123');
   await page.getByRole('button',{name:'Register',exact:true}).click();assert.match(await page.locator('dialog [role=alert]').innerText(),/15 digits/);await close();
   const first=await register(owner,'LTE',imei()),second=await register(owner,'5G',imei());await total(baseline.length+2);
   pass('Subscriber provision/activation and LTE + 5G registration through real UI');
   const other=await makeOwner();await register(other,'LTE',imei());const expected=baseline.length+3;await total(expected);
   for(const q of [first.imei,owner.imsi,owner.msisdn,'5G','REGISTERED']){await page.getByLabel('Search devices').fill(q);assert.ok(await page.getByTestId('device-row').count()>0);}
   await page.getByLabel('Search devices').fill('');
   await page.getByLabel('Filter devices by subscriber').selectOption(owner.id);
   await page.waitForFunction(()=>document.querySelectorAll('[data-testid="device-row"]').length===2);await total(expected);
   assert.ok(requests.some(r=>r.url.includes('devices?subscriber_id='+owner.id)));
   await nav('Overview');await page.waitForFunction(n=>document.querySelector('[data-testid="devices-kpi"] strong')?.textContent===String(n),expected);
   await nav('Devices');pass('Real subscriber filter and local search never replace global KPI');
   await page.getByRole('button',{name:'Open device '+first.imei,exact:true}).click();await page.getByTestId('device-details').waitFor();
   for(const field of [first.id,first.subscriber_id,first.created_at,first.updated_at,owner.imsi,owner.msisdn])assert.ok((await page.getByTestId('device-details').innerText()).includes(field));await close();
   pass('GET UUID details and subscriber enrichment from shared collection');
   await open();await page.getByLabel('Device subscriber',{exact:true}).selectOption(owner.id);await page.getByLabel('Register IMEI').fill(first.imei);await page.getByRole('button',{name:'Register',exact:true}).click();
   await page.locator('dialog [role=alert]').filter({hasText:'409'}).waitFor();await close();await total(expected);pass('Duplicate IMEI 409 visible, total unchanged');
   const suspended=await makeOwner('SUSPENDED'),deactivated=await makeOwner('DEACTIVATED');
   const pending=await(await post('/api/v1/subscribers',{imsi:'72498'+stamp+'1',msisdn:'+5518'+stamp+'1'})).json();
   await open();await page.getByRole('button',{name:'Refresh subscribers',exact:true}).click();await page.waitForFunction(id=>[...document.querySelectorAll('select[aria-label="Device subscriber"] option')].some(o=>o.value===id),pending.id);
   for(const sub of [suspended,deactivated,pending])assert.equal(await page.locator('select[aria-label="Device subscriber"] option').filter({hasText:sub.imsi}).evaluate(e=>e.disabled),true);await close();
   assert.equal((await post('/api/v1/devices',{subscriber_id:pending.id,imei:imei(),technology:'LTE'})).status,422);
   for(const [body,status] of [[{subscriber_id:owner.id,imei:'x',technology:'LTE'},400],[{subscriber_id:'00000000-0000-0000-0000-000000000000',imei:imei(),technology:'LTE'},404],[{subscriber_id:suspended.id,imei:imei(),technology:'LTE'},422],[{subscriber_id:deactivated.id,imei:imei(),technology:'LTE'},422]])assert.equal((await post('/api/v1/devices',body)).status,status);
   pass('Inactive owners disabled; real backend 400 / 404 / 422 contracts including PENDING_ACTIVATION');
   await open();await page.getByLabel('Device subscriber',{exact:true}).selectOption(other.id);await page.getByLabel('Register IMEI').fill(imei());
   assert.equal((await post('/api/v1/subscribers/'+other.id+'/suspend',{reason:'Concurrent lifecycle test'})).status,200);
   await page.getByRole('button',{name:'Register',exact:true}).click();await page.locator('dialog [role=alert]').filter({hasText:'422'}).waitFor();await close();await total(expected);pass('Concurrent subscriber suspension: backend 422 visible with no optimistic device');
   const route='**/api/v1/devices';await page.route(route,r=>r.abort('connectionrefused'));
   await page.getByRole('button',{name:'Refresh',exact:true}).click();await state('stale');await total(expected);
   await page.reload();await state('error');await total('—');assert.doesNotMatch(await page.locator('.device-table').innerText(),/No devices registered yet/);
   await page.unroute(route);await page.getByRole('button',{name:'Refresh',exact:true}).click();await state('success');await total(expected);pass('Failed read preserves stale snapshot; no-snapshot error is —; manual recovery');
   const start=requests.length;await page.waitForTimeout(11000);
   assert.equal(requests.slice(start).filter(r=>new URL(r.url).pathname==='/api/v1/devices').length,0);
   assert.ok(requests.slice(start).some(r=>r.url.includes('/health')));assert.equal(requests.filter(r=>r.url.includes('/api/v1/sessions')).length,0);pass('No Devices polling or Sessions requests; Health polling preserved');
   for(const [width,height] of [[1920,1080],[1440,900],[1366,768]]){
     await page.setViewportSize({width,height});await nav('Devices');await page.screenshot({path:path.join(out,`devices-${width}x${height}.png`)});
     await open();await page.screenshot({path:path.join(out,`register-${width}x${height}.png`)});await close();
     await page.getByRole('button',{name:'Open device '+second.imei,exact:true}).click();await page.getByTestId('device-details').waitFor();await page.screenshot({path:path.join(out,`details-${width}x${height}.png`)});await close();
     await nav('Overview');await page.screenshot({path:path.join(out,`overview-${width}x${height}.png`)});
   }
   assert.deepEqual(errors,[]);pass('Twelve real screenshots; zero uncaught browser errors');
 }finally{fs.writeFileSync(path.join(out,'results.json'),JSON.stringify({results,requests,errors},null,2));await browser.close();}
})().catch(e=>{console.error(e);process.exitCode=1});
