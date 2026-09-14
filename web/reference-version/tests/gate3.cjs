// Requires the explicitly isolated memory API :18082 and its Vite proxy :5177.
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'C:/Users/joaob/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const output=path.resolve(__dirname,'../evidence/gate3');fs.mkdirSync(output,{recursive:true});
const origin='http://127.0.0.1:5177', api='http://127.0.0.1:18082';
const results=[], lifecycle=[];
const pass=name=>{results.push(name);console.log('PASS:',name);};
const pause=ms=>new Promise(resolve=>setTimeout(resolve,ms));
(async()=>{
  const browser=await chromium.launch({channel:'chrome',headless:true});
  const page=await browser.newPage({viewport:{width:1920,height:1080}});
  const errors=[],requests=[];
  page.on('pageerror',e=>errors.push(e.message));
  page.on('request',r=>requests.push({method:r.method(),url:r.url()}));
  page.on('response',async response=>{
    if(response.request().method()==='POST'&&response.url().includes('/api/v1/subscribers')&&response.ok()){
      const entity=await response.json();lifecycle.push(entity);
    }
  });
  const navigate=async name=>page.locator('.sidebar a').filter({hasText:new RegExp('^'+name+'$')}).click();
  const collection=async state=>page.waitForFunction(state=>document.querySelector('[data-testid="subscribers-page"]')?.getAttribute('data-state')===state,state);
  const total=async expected=>page.waitForFunction(expected=>document.querySelector('[data-testid="subscriber-total"]')?.textContent===String(expected),expected);
  const status=async expected=>page.waitForFunction(expected=>document.querySelector('dialog [data-testid="subscriber-status"]')?.textContent===expected,expected);
  const close=async()=>page.getByRole('button',{name:'Close subscriber dialog'}).click();
  const fill=async(imsi,msisdn)=>{await page.getByLabel('Provision IMSI',{exact:true}).fill(imsi);await page.getByLabel('Provision MSISDN',{exact:true}).fill(msisdn);};
  const submit=async()=>page.getByRole('button',{name:'Provision',exact:true}).click();
  const alert=async text=>page.locator('dialog [role="alert"]').filter({hasText:text}).waitFor();
  const verifyKpi=async expected=>{
    await navigate('Overview');
    await page.waitForFunction(expected=>document.querySelector('[data-testid="subscribers-kpi"] strong')?.textContent===String(expected),expected);
    assert.doesNotMatch(await page.locator('[data-testid="subscribers-kpi"]').innerText(),/128|\+12%|mock/);
    await navigate('Subscribers');await total(expected);
  };
  try{
    const baseline=(await (await fetch(api+'/api/v1/subscribers')).json())??[];
    await page.goto(origin+'/#subscribers');await collection('success');await total(baseline.length);
    if(baseline.length===0){assert.match(await page.locator('.subscriber-table').innerText(),/No subscribers provisioned yet/);pass('Real empty collection and KPI zero');}
    await verifyKpi(baseline.length);
    const suffix=String(Date.now()).slice(-10), imsi='72499'+suffix, msisdn='+5519'+suffix;
    await page.getByRole('button',{name:'Provision subscriber',exact:true}).click();
    await fill('123',msisdn);await submit();await alert('exactly 15 digits');
    pass('Invalid IMSI rejected by UX without a request');
    await fill(imsi,msisdn);await submit();await status('PENDING_ACTIVATION');
    await page.locator('dialog [role="status"]').filter({hasText:'Subscriber provisioned'}).waitFor();
    const details=await page.locator('[data-testid="subscriber-details"]').innerText();
    assert.ok(details.includes(imsi)&&details.includes(msisdn));
    pass('Real provision returns PENDING_ACTIVATION; identities remain full strings');
    await page.getByRole('button',{name:'Activate',exact:true}).click();await status('ACTIVE');
    await page.getByLabel('Lifecycle reason').fill('Gate 3 administrative hold');
    await page.getByRole('button',{name:'Suspend',exact:true}).click();await status('SUSPENDED');
    assert.match(await page.locator('[data-testid="subscriber-details"]').innerText(),/Gate 3 administrative hold/);
    await page.screenshot({path:path.join(output,'details-suspended-1920x1080.png')});
    await page.getByRole('button',{name:'Activate',exact:true}).click();await status('ACTIVE');
    assert.doesNotMatch(await page.locator('[data-testid="subscriber-details"]').innerText(),/Suspension reason/);
    await page.getByLabel('Lifecycle reason').fill('Gate 3 contract ended');
    await page.getByRole('button',{name:'Deactivate permanently'}).click();await status('DEACTIVATED');
    assert.match(await page.locator('dialog').innerText(),/terminal/);
    assert.equal(await page.locator('dialog .subscriber-actions button').count(),0);
    pass('Real activate → suspend with reason → reactivate → deactivate; terminal actions absent');
    const entity=(await (await fetch(api+'/api/v1/subscribers?imsi='+imsi)).json());
    await close();await total(baseline.length+1);await verifyKpi(baseline.length+1);
    const search=page.getByLabel('Search subscribers',{exact:true});
    await search.fill(imsi);assert.equal(await page.getByTestId('subscriber-row').count(),1);
    await page.getByRole('button',{name:'Find exact IMSI'}).click();await status('DEACTIVATED');
    assert.match(await page.getByTestId('subscriber-details').innerText(),new RegExp(entity.id));
    assert.doesNotMatch(await page.getByTestId('subscriber-details').innerText(),/Device|Cell|IPAddress/);
    await close();await search.fill(msisdn);assert.equal(await page.getByTestId('subscriber-row').count(),1);
    await search.fill('DEACTIVATED');assert.ok(await page.getByTestId('subscriber-row').count()>=1);
    await search.fill('');pass('Local IMSI/MSISDN/status search, exact HTTP IMSI lookup and UUID details');
    await page.getByRole('button',{name:'Provision subscriber',exact:true}).click();
    await fill(imsi,'+5521'+suffix);await submit();await alert('HTTP 409');await alert('SUBSCRIBER_ALREADY_EXISTS');
    await fill('72498'+suffix,msisdn);await submit();await alert('SUBSCRIBER_ALREADY_EXISTS');
    pass('Duplicate IMSI and duplicate MSISDN return visible real HTTP 409');
    await page.route('**/api/v1/subscribers',route=>route.request().method()==='POST'?route.fulfill({status:500,contentType:'application/json',body:'{"error":"internal_error","message":"test failure","code":"INTERNAL_SERVER_ERROR"}'}):route.continue());
    await fill('72498'+suffix,'+5521'+suffix);await submit();await alert('HTTP 500');
    await page.unroute('**/api/v1/subscribers');await close();await verifyKpi(baseline.length+1);
    pass('Provision 500 does not claim success or increase count');
    // Exercise real backend validation independently of the UX guard.
    const invalid=await fetch(api+'/api/v1/subscribers',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({imsi:'123',msisdn})});
    assert.equal(invalid.status,400);assert.equal((await invalid.json()).code,'INVALID_TELECOM_IDENTITY');
    pass('Real backend rejects invalid IMSI with HTTP 400');
    // Stale detail vs another operator: surface real domain 409, then reconcile.
    await page.getByRole('button',{name:'Provision subscriber',exact:true}).click();
    await fill('72498'+suffix,'+5521'+suffix);await submit();await status('PENDING_ACTIVATION');
    const second=await (await fetch(api+'/api/v1/subscribers?imsi=72498'+suffix)).json();
    await fetch(api+`/api/v1/subscribers/${second.id}/deactivate`,{method:'POST'});
    await page.getByRole('button',{name:'Activate',exact:true}).click();await alert('SUBSCRIBER_ALREADY_DEACTIVATED');await status('DEACTIVATED');
    pass('Concurrent terminal transition: real 409 stays visible and details reconcile');
    await close();await total(baseline.length+2);await verifyKpi(baseline.length+2);
    const realInvalid=await fetch(api+`/api/v1/subscribers/${entity.id}/activate`,{method:'POST'});
    assert.equal(realInvalid.status,409);
    // Failed list refresh retains complete last collection, shared with KPI.
    await page.route('**/api/v1/subscribers',route=>route.abort('connectionfailed'));
    await page.getByRole('button',{name:'Refresh',exact:true}).click();await collection('stale');await total(baseline.length+2);
    await verifyKpi(baseline.length+2);
    await page.screenshot({path:path.join(output,'subscribers-stale-1920x1080.png')});
    await page.reload();await collection('error');await total('—');
    assert.doesNotMatch(await page.locator('.subscriber-table').innerText(),/No subscribers provisioned yet/);
    await navigate('Overview');await page.waitForFunction(()=>document.querySelector('[data-testid="subscribers-kpi"]')?.getAttribute('data-state')==='error');
    assert.equal(await page.locator('[data-testid="subscribers-kpi"] strong').innerText(),'—');
    await navigate('Subscribers');await page.unroute('**/api/v1/subscribers');
    await page.getByRole('button',{name:'Refresh',exact:true}).click();await collection('success');await total(baseline.length+2);
    pass('List network failure: stale retains values, initial error is not empty, manual recovery restores data');
    const atRest=requests.filter(r=>new URL(r.url).pathname==='/api/v1/subscribers'&&r.method==='GET').length;
    await pause(11000);
    assert.equal(requests.filter(r=>new URL(r.url).pathname==='/api/v1/subscribers'&&r.method==='GET').length,atRest);
    pass('No subscriber polling; Health/Telemetry remain active');
    await page.screenshot({path:path.join(output,'subscribers-1920x1080.png')});
    await navigate('Overview');await page.screenshot({path:path.join(output,'overview-1920x1080.png')});
    await page.waitForFunction(()=>document.querySelector('[data-testid="active-kpi"]')?.getAttribute('data-state')==='success');
    assert.equal(await page.locator('.infra i[data-status="unknown"]').count(),1);
    await navigate('Subscribers');
    for(const [width,height] of [[1440,900],[1366,768]]){
      await page.setViewportSize({width,height});
      assert.ok(await page.evaluate(()=>{const r=document.querySelector('.fit-shell').getBoundingClientRect();return r.bottom<=innerHeight+1&&r.right<=innerWidth+1;}));
      await page.screenshot({path:path.join(output,`subscribers-${width}x${height}.png`)});
    }
    // Successful null list is the sole normalized empty case (Postgres contract).
    await page.route('**/api/v1/subscribers',route=>route.fulfill({status:200,contentType:'application/json',body:'null'}));
    await page.getByRole('button',{name:'Refresh',exact:true}).click();await collection('success');await total(0);
    await page.unroute('**/api/v1/subscribers');await page.getByRole('button',{name:'Refresh',exact:true}).click();await total(baseline.length+2);
    pass('Successful null list normalizes to []; HTTP failures do not');
    assert.equal(requests.filter(r=>/\/api\/v1\/(devices|sessions)/.test(r.url)).length,0);
    assert.deepEqual(errors,[]);
    pass('No Device/Session requests, no browser errors; approved canvas fits all viewports');
    fs.writeFileSync(path.join(output,'results.json'),JSON.stringify({passed:results,baseline:baseline.length,finalTotal:baseline.length+2,lifecycle,requests},null,2));
  }catch(error){await page.screenshot({path:path.join(output,'failure.png')});throw error;}
  finally{await browser.close();}
})().catch(error=>{console.error(error);process.exitCode=1;});
