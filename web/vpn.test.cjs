const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');
const source = fs.readFileSync(path.join(__dirname, 'static/vpn.js'), 'utf8');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');

function harness(response) {
  const elements = {};
  function element() { return {textContent:'', value:'', disabled:false, style:{}, children:[], hidden:false,
    classList:{add(){},remove(){},toggle(){}},
    focus(){context.document.activeElement=this;},
    replaceChildren(...items){this.children=items;}, append(...items){this.children.push(...items);}}; }
  for (const match of html.matchAll(/id="([^"]+)"/g)) {
    elements[match[1]] = element();
  }
  const calls = [];
  const context = vm.createContext({
    currentPage:'devices', currentUser: {role: 'admin'}, devices: [{id: 'pc', hostname: '<PC>', online: true, os: 'windows', version: '0.2.69'}],
    document: {getElementById: id => elements[id], createElement: element, querySelectorAll:()=>[], activeElement: {focus() {}}},
    api: async (url, options) => { calls.push({url, options}); return typeof response === 'function' ? response(url, options) : response; },
    loadDevices(){}, loadStats(){}, loadBranchAssets(){}, updateRemoteDeviceList(){},
    setInterval() { calls.push('poll'); return 1; }, clearInterval() { calls.push('stop'); },
    showToast(message) { calls.push(message); }, appConfirm: async () => true,
    esc: value => String(value)
  });
  vm.runInContext(source, context);
  const app=fs.readFileSync(path.join(__dirname,'static/app.js'),'utf8');
  vm.runInContext(app.slice(app.indexOf('function showPage(page)'),app.indexOf('var lastStats = null')),context);
  return {context, elements, calls};
}

test('sidebar master owns settings and device modal remains connection-only', () => {
  const master=html.slice(html.indexOf('<section id="page-vpn"'),html.indexOf('<!-- REMOTE DESKTOP PAGE -->'));
  const modal=html.slice(html.indexOf('<div id="vpnModal"'));
  for (const id of ['vpnSelfEnabled','vpnSelfRoute','vpnAdvertiseLAN','vpnLANSubnet','vpnLANClient']) {
    assert.ok(master.includes('id="'+id+'"'));
    assert.ok(!modal.includes('id="'+id+'"'));
    assert.equal(html.split('id="'+id+'"').length,2);
  }
  assert.match(html,/id="navVPNMaster" data-page="vpn" onclick="showPage\('vpn'\)"/);
  assert.match(modal,/Kelola di Master VPN/);
});

test('master paginates independently, preserves drafts and resolves off-page devices', async () => {
  const listed={id:'listed',hostname:'<script>bad</script>',os:'windows',online:true};
  const {context,elements,calls}=harness(url=>{
    if(url.startsWith('/api/devices?')) return {devices:[listed],total:60};
    if(url==='/api/devices/outside') return {id:'outside',hostname:'Outside',os:'windows',online:true};
    if(url.startsWith('/api/vpn/self-service')) return {enabled:false,route_lan:''};
    return {ready:true,sessions:{}};
  });
  await context.showPage('vpn');
  assert.equal(elements.vpnMasterRows.children[0].children[0].children[0].textContent,listed.hostname);
  assert.equal(elements.vpnMasterNext.disabled,false);
  elements.vpnSelfEnabled.checked=true;
  elements.vpnSelfRoute.value='192.168.10.0/24';
  await context.loadVPNMaster(25);
  assert.ok(calls.some(call=>call.url&&call.url.includes('offset=25&limit=25')));
  assert.equal(elements.vpnSelfRoute.value,'192.168.10.0/24');
  assert.equal(elements.vpnSelfEnabled.checked,true);
  assert.equal(context.devices[0].id,'pc');
  elements.vpnMasterSearch.value='branch & laptop';
  await context.loadVPNMaster(0);
  assert.ok(calls.some(call=>call.url&&call.url.includes('search=branch%20%26%20laptop')));
  await context.openVPNMaster('outside');
  assert.equal(elements.vpnMasterSelectedName.textContent,'Outside');
  assert.ok(calls.some(call=>call.url==='/api/devices/outside'));
  assert.equal(calls.filter(call=>call.options).length,0);
});

test('modal never applies master gateway draft and keeps its own target', async () => {
  const {context,elements,calls}=harness({ready:true,sessions:{}});
  await context.selectVPNMasterDevice('pc');
  elements.vpnAdvertiseLAN.checked=true;
  elements.vpnLANSubnet.value='192.168.1.0/24';
  elements.vpnLANClient.value='other';
  context.devices.push({id:'other',os:'windows',online:true});
  await context.openVPNModal('other');
  assert.equal(elements.vpnAdvertiseLAN.checked,true);
  await context.triggerVPN('connect');
  assert.deepEqual(JSON.parse(calls.find(call=>call.options).options.body),{device:'other',operation:'connect'});
  assert.equal(elements.vpnLANSubnet.value,'192.168.1.0/24');
  assert.equal(calls.filter(call=>call.url&&call.url.includes('self-service')).length,1);
});

test('master navigation is admin-only and polling stops after leaving both views', async () => {
  const {context,calls}=harness(url=>url.startsWith('/api/devices?')?{devices:[],total:0}:{ready:true,sessions:{}});
  context.currentUser.role='viewer';
  await context.showPage('vpn');
  await context.openVPNMaster('pc');
  assert.equal(context.currentPage,'devices');
  assert.equal(calls.filter(call=>call.url).length,0);
  context.currentUser.role='admin';
  await context.showPage('vpn');
  assert.equal(calls.filter(call=>call==='poll').length,1);
  await context.openVPNModal('pc');
  context.closeVPNModal();
  assert.equal(calls.filter(call=>call==='stop').length,0);
  context.showPage('devices');
  assert.equal(calls.filter(call=>call==='stop').length,1);
});

test('self-service policy requires loaded admin state and confirmation', async () => {
  const {context, elements, calls} = harness({enabled: false, route_lan: ''});
  await context.selectVPNMasterDevice('pc');
  assert.equal(elements.vpnSelfEnabled.checked, false);
  assert.equal(elements.btnVPNSelfSave.disabled, false);
  elements.vpnSelfEnabled.checked = true;
  elements.vpnSelfRoute.value = '192.168.1.0/24';
  context.appConfirm = async () => false;
  await context.saveVPNSelfPolicy();
  assert.equal(calls.filter(call => call.options).length, 0);
  context.appConfirm = async () => true;
  await context.saveVPNSelfPolicy();
  const saved = calls.find(call => call.options);
  assert.equal(saved.url, '/api/vpn/self-service?device=pc');
  assert.equal(saved.options.method, 'PUT');
  assert.deepEqual(JSON.parse(saved.options.body), {enabled:true, route_lan:'192.168.1.0/24'});
  context.currentUser.role = 'viewer';
  await context.saveVPNSelfPolicy();
  assert.equal(calls.filter(call => call.options).length, 1);
});

test('unavailable policy and device switches disable stale policy writes', async () => {
  const {context, elements, calls} = harness(null);
  await context.selectVPNMasterDevice('pc');
  assert.equal(elements.btnVPNSelfSave.disabled, true);
  await context.saveVPNSelfPolicy();
  assert.equal(calls.filter(call => call.options).length, 0);
  let finish;
  context.api = () => new Promise(resolve => { finish = resolve; });
  const loading = context.loadVPNSelfPolicy();
  vm.runInContext('vpnMasterSelectedDevice = "other"', context);
  finish({enabled:true, route_lan:'192.168.1.0/24'});
  await loading;
  assert.equal(elements.btnVPNSelfSave.disabled, true);
  assert.equal(elements.vpnSelfEnabled.checked, false);
});

test('VPN action opens real dialog and shows readiness without connecting', async () => {
  const {context, elements, calls} = harness({ready: false, detail: 'Hub belum dikonfigurasi', sessions: {}});
  await context.openVPNModal('pc');
  assert.equal(elements.vpnModal.style.display, 'flex');
  assert.equal(elements.vpnHubDetail.textContent, 'Hub belum dikonfigurasi');
  assert.equal(elements.btnVPNConnect.disabled, true);
  assert.equal(calls.filter(call => call.options && call.options.method === 'POST').length, 0);
  context.closeVPNModal();
  assert.equal(elements.vpnModal.style.display, 'none');
  assert.ok(calls.includes('stop'));
});

test('VPN API failure is visible and prevents connect', async () => {
  const {context, elements} = harness({error: 'VPN pilot belum tersedia'});
  await context.openVPNModal('pc');
  assert.equal(elements.vpnHubDetail.textContent, 'VPN pilot belum tersedia');
  assert.equal(elements.vpnClientStatus.textContent, 'UNKNOWN');
  assert.equal(elements.btnVPNConnect.disabled, true);
});

test('VPN connect sends selected device only after confirmation', async () => {
  const {context, calls} = harness({ready: true, sessions: {}});
  await context.openVPNModal('pc');
  context.appConfirm = async () => false;
  await context.triggerVPN('connect');
  assert.equal(calls.filter(call => call.options).length, 0);
  context.appConfirm = async () => true;
  await context.triggerVPN('connect');
  const sent = calls.find(call => call.options);
  assert.deepEqual(JSON.parse(sent.options.body), {device: 'pc', operation: 'connect'});
});

test('preparing and connecting sessions remain cancellable', async () => {
  for (const state of ['preparing','connecting']) {
    const {context,elements,calls} = harness({ready:true,sessions:{pc:{state,detail:'waiting'}}});
    await context.openVPNModal('pc');
    assert.equal(elements.btnVPNConnect.disabled,true);
    assert.equal(elements.btnVPNDisconnect.disabled,false);
    await context.triggerVPN('disconnect');
    assert.deepEqual(JSON.parse(calls.find(call=>call.options).options.body),{device:'pc',operation:'disconnect'});
  }
});

test('LAN opt-in requires client, subnet and explicit confirmation', async () => {
  const {context,elements,calls} = harness({ready:true,sessions:{}});
  await context.refreshVPNState();
  await context.selectVPNMasterDevice('pc');
  assert.equal(elements.vpnAdvertiseLAN.checked,false);
  assert.equal(elements.vpnLANSubnet.disabled,true);
  elements.vpnAdvertiseLAN.checked=true;
  context.renderVPNLANControls();
  assert.equal(elements.vpnLANSubnet.disabled,false);
  await context.triggerVPN('connect',true);
  assert.equal(calls.filter(call=>call.options).length,0);
  elements.vpnLANSubnet.value='192.168.1.0/24';elements.vpnLANClient.value='other-page-device';
  let confirmation='';
  context.appConfirm=async message=>{confirmation=message;return false;};
  await context.triggerVPN('connect',true);
  assert.equal(calls.filter(call=>call.options).length,0);
  assert.match(confirmation,/192\.168\.1\.0\/24/);
  assert.match(confirmation,/other-page-device/);
  context.appConfirm=async()=>true;
  await context.triggerVPN('connect',true);
  assert.deepEqual(JSON.parse(calls.find(call=>call.options).options.body),{device:'pc',operation:'connect',advertise_lan:'192.168.1.0/24',lan_clients:['other-page-device']});
});

test('client confirms assigned LAN route and active gateway controls stay locked', async () => {
  const {context,elements,calls}=harness({ready:true,sessions:{gateway:{state:'connected',advertise_lan:'192.168.1.0/24',lan_clients:['pc']}}});
  await context.openVPNModal('pc');
  await context.triggerVPN('connect');
  assert.equal(JSON.parse(calls.find(call=>call.options).options.body).route_lan,'192.168.1.0/24');
  context.devices.push({id:'gateway',os:'windows',online:true});
  await context.selectVPNMasterDevice('gateway');
  assert.equal(elements.vpnAdvertiseLAN.checked,true);
  assert.equal(elements.vpnAdvertiseLAN.disabled,true);
  assert.equal(elements.vpnLANSubnet.disabled,true);
  assert.match(elements.vpnLANStatus.textContent,/Gateway LAN/);
  context.appConfirm=async()=>false;
  const before=calls.filter(call=>call.options).length;
  await context.triggerVPN('disconnect',true);
  assert.equal(calls.filter(call=>call.options).length,before);
});

test('switching device clears LAN opt-in and denied roles cannot open VPN', async () => {
  const {context,elements,calls}=harness({ready:true,sessions:{}});
  await context.selectVPNMasterDevice('pc');
  elements.vpnAdvertiseLAN.checked=true;elements.vpnLANSubnet.value='192.168.1.0/24';
  context.devices.push({id:'other',os:'windows',online:true});
  await context.selectVPNMasterDevice('other');
  assert.equal(elements.vpnAdvertiseLAN.checked,false);
  assert.equal(elements.vpnLANSubnet.value,'');
  const before=calls.length;context.currentUser.role='viewer';
  await context.openVPNModal('pc');
  assert.equal(calls.slice(before).filter(call=>call.url).length,0);
});
