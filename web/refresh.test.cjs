const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const test = require('node:test');
const source = fs.readFileSync(require('node:path').join(__dirname,'static/app.js'),'utf8').replaceAll(String.fromCharCode(13),'');

test('polling only refreshes visible page and leaves active remote untouched', async () => {
  const calls=[];
  const context=vm.createContext({document:{hidden:false},currentPage:'remote',remoteWS:{},
    loadStats:()=>calls.push('stats'),loadDevices:()=>calls.push('devices'),loadVPNMaster:()=>calls.push('vpn'),
    loadBranchAssets:async()=>{calls.push('assets');},renderAssets:()=>calls.push('render')});
  vm.runInContext(source.slice(source.indexOf('function refreshVisiblePage()'),source.indexOf('// ==================== API')),context);
  context.refreshVisiblePage();
  assert.deepEqual(calls,[]);
  context.currentPage='dashboard';context.document.hidden=true;
  context.refreshVisiblePage();
  assert.deepEqual(calls,[]);
  context.document.hidden=false;context.refreshVisiblePage();
  assert.deepEqual(calls,['stats','devices']);calls.length=0;
  context.currentPage='devices';context.refreshVisiblePage();
  assert.deepEqual(calls,['devices']);calls.length=0;
  context.currentPage='assets';context.refreshVisiblePage();await Promise.resolve();
  assert.deepEqual(calls,['assets','render']);
  calls.length=0;context.currentPage='vpn';context.refreshVisiblePage();
  assert.deepEqual(calls,['vpn']);
});

test('device metadata save includes dedicated identity fields', async () => {
  let payload;
  const values={modalTags:'office',modalGroup:'HO',modalNote:'note',modalSerialNumber:' SN-123 ',modalProductID:' PROD-456 ',modalAcquisitionYear:'2024'};
  const context=vm.createContext({currentDevice:{id:'pc'},document:{getElementById:id=>({value:values[id]})},
    api:async(url,options)=>{payload=JSON.parse(options.body);return {status:'ok'};},
    closeDeviceModal(){},loadDevices(){},loadGroups(){},loadBranchAssets(){},showToast(){},appAlert(){throw Error('save failed');}});
  const start=source.indexOf('async function saveDeviceMeta()');
  const newline=String.fromCharCode(10);
  vm.runInContext(source.slice(start,start+source.slice(start).indexOf(newline+'}'+newline)+3),context);
  await context.saveDeviceMeta();
  assert.equal(payload.serial_number,'SN-123');assert.equal(payload.product_id,'PROD-456');
  assert.equal(payload.note,'note');assert.equal(payload.tags,'office');
});
