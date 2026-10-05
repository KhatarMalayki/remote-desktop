const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const http = require('node:http');
const { createRequire } = require('node:module');
const { chromium } = process.env.PLAYWRIGHT_MODULES ? createRequire(path.join(process.env.PLAYWRIGHT_MODULES, 'package.json'))('playwright') : require('playwright');
const root = path.join(__dirname, '..');
const output = process.env.UI_SMOKE_OUTPUT || path.join(root, 'output', 'playwright');
const fixture = {id:'test-workstation', hostname:'WORKSTATION-01', assigned_to:'Pengguna Pengujian', online:true, os:'windows', arch:'amd64', version:'0.2.58', cpu_cores:8, memory_total:16, memory_used:8, disk_total:512, disk_used:128, local_ip:'192.0.2.10', branch:'Cabang Pengujian', last_seen:Math.floor(Date.now()/1000), endpoint:{applications:[{label:'ME',status:'unknown'}]}};
const responses = {
  '/api/users': Array.from({length:18}, (_, index) => ({id:index+1,username:'demo_department_nama_panjang_'+index,role:'spv',branch:'Bintaro - Surya Sudeco, Bandung - Surya Sudeco',mfa_enabled:true})),
  '/api/agent/version': {version:'0.2.58'},
  '/api/auth/login': {token:'local-fixture-only'},
  '/api/auth/me': {id:1,username:'test-admin',role:'admin',mfa_enabled:true},
  '/api/auth/mfa/status': {mfa_enabled:true},
  '/api/stats': {total_devices:1,online_devices:1,offline_devices:0,os_distribution:{windows:1}},
  '/api/devices': {devices:[fixture],total:1,offset:0,limit:50,server_version:'0.2.58'},
  '/api/groups': ['Cabang Pengujian'],
  '/api/vpn/pilot': {ready:false, detail:'Hub VPN belum dikonfigurasi', network:'', sessions:{}},
  '/api/devices/test-workstation': fixture
};
(async () => {
  fs.mkdirSync(output, {recursive:true});
  const server = http.createServer((request,response) => {
    const pathname = new URL(request.url,'http://localhost').pathname;
    const filename = path.resolve(root,'web','.'+(pathname==='/'?'/index.html':pathname));
    if (!filename.startsWith(path.resolve(root,'web')+path.sep) || !fs.existsSync(filename) || !fs.statSync(filename).isFile()) {response.writeHead(404).end();return;}
    response.setHeader('Content-Type',filename.endsWith('.css')?'text/css':filename.endsWith('.js')?'text/javascript':'text/html');
    response.end(fs.readFileSync(filename));
  });
  await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
  let browser;
  try {
    browser = await chromium.launch({headless:true,...(process.env.BROWSER_EXECUTABLE?{executablePath:process.env.BROWSER_EXECUTABLE}:{}),args:['--disable-gpu','--no-sandbox']});
    const results = [];
    for (const width of [1440,390]) {
      const page = await browser.newPage({viewport:{width,height:1000}});
      const errors = [];
      page.on('pageerror', error=>errors.push(error.message));
      await page.route('**/api/**',route=>route.fulfill({json:responses[new URL(route.request().url()).pathname] || []}));
      await page.goto('http://127.0.0.1:'+server.address().port,{waitUntil:'domcontentloaded'});
      await page.screenshot({path:path.join(output,'login-'+width+'.png')});
      await page.locator('#loginUser').fill('test-admin');
      await page.locator('#loginPass').fill('fixture-not-a-password');
      await page.getByRole('button',{name:'Sign In',exact:true}).click();
      await page.locator('#appContainer').waitFor({state:'visible'});
      for (const section of ['dashboard','branch-assets','devices','assets','remote']) {
        await page.locator('.nav-item[data-page="'+section+'"]').click();
        await page.locator('#page-'+section).waitFor({state:'visible'});
        if (section === 'devices') {
          assert.equal(await page.locator('#devicesTable tbody tr').count(), 1, 'Device fixture must render');
          assert.equal(await page.locator('#devicesTable thead th').count(), 6);
          const remoteLink = page.locator('#devicesTable a[target="_blank"]');
          assert.equal(await remoteLink.getAttribute('rel'), 'noopener noreferrer');
          assert.equal(await remoteLink.getAttribute('href'), '#remote=test-workstation');
          await page.locator('#devicesTable').getByRole('button',{name:'VPN',exact:true}).click();
          await page.locator('#vpnModal').waitFor({state:'visible'});
          await page.getByText('Hub VPN belum dikonfigurasi',{exact:true}).waitFor();
          assert.equal(await page.locator('#btnVPNConnect').isDisabled(),true);
          assert.equal(await page.locator('#vpnDevicePicker').inputValue(),'test-workstation');
          await page.screenshot({path:path.join(output,'vpn-'+width+'.png')});
          await page.locator('#vpnDevicePicker').press('Shift+Tab');
          assert.equal(await page.locator('#vpnModal button').last().evaluate(element=>element===document.activeElement),true);
          await page.keyboard.press('Escape');
          await page.locator('#vpnModal').waitFor({state:'hidden'});
        }
        const overflow = await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1);
        results.push({width,section,overflow});
        await page.screenshot({path:path.join(output,section+'-'+width+'.png')});
      }
      const modals = await page.locator('.modal-overlay:has(.modal)').evaluateAll(elements=>elements.map(element=>element.id).filter(Boolean));
      for (const id of modals) {
        await page.evaluate(id=>{document.getElementById(id).style.display='flex';},id);
        const bounds = await page.locator('#'+id+' .modal').first().boundingBox();
        results.push({width,modal:id,fits:!!bounds && bounds.x>=-1 && bounds.x+bounds.width<=width+1});
        await page.evaluate(id=>{document.getElementById(id).style.display='none';},id);
      }
      await page.locator('#navUsers').click();
      await page.locator('#usersModal').waitFor({state:'visible'});
      await page.locator('#usersTableContainer tbody tr').first().waitFor();
      assert.equal(await page.locator('#usersTableContainer tbody tr').count(),18);
      assert.equal(await page.locator('#usersModal .modal').evaluate(element=>element.scrollWidth>element.clientWidth+1),false,'Account modal must not overflow horizontally');
      await page.getByRole('button',{name:'Tutup kelola akun pengguna',exact:true}).click();
      await page.locator('#usersModal').waitFor({state:'hidden'});
      await page.locator('#navUsers').click();
      await page.screenshot({path:path.join(output,'administration-'+width+'.png')});
      assert.deepEqual(errors,[], 'Browser runtime errors');
      await page.close();
    }
    fs.writeFileSync(path.join(output,'redesign-smoke.json'),JSON.stringify(results,null,2));
    const failures = results.filter(result=>result.overflow || result.fits===false);
    assert.deepEqual(failures,[], 'Layout overflow');
    console.log(results.length+' page/modal layout checks passed; mocked API, no production actions.');
  } finally { if(browser) await browser.close(); await new Promise(resolve=>server.close(resolve)); }
})().catch(error=>{console.error(error);process.exitCode=1;});
