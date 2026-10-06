const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');
const source = fs.readFileSync(path.join(__dirname, 'static/vpn.js'), 'utf8');
const html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');

function harness(response) {
  const elements = {};
  for (const match of html.matchAll(/id="([^"]+)"/g)) {
    elements[match[1]] = {textContent: '', disabled: false, style: {}, open: false,
      showModal() { this.open = true; }, close() { this.open = false; }, focus() {},
      replaceChildren() {}, append() {}};
  }
  const calls = [];
  const context = vm.createContext({
    currentUser: {role: 'admin'}, devices: [{id: 'pc', hostname: '<PC>', online: true, os: 'windows', version: '0.2.65'}],
    document: {getElementById: id => elements[id], createElement: () => ({}), activeElement: {focus() {}}},
    api: async (url, options) => { calls.push({url, options}); return response; },
    setInterval() { calls.push('poll'); return 1; }, clearInterval() { calls.push('stop'); },
    showToast(message) { calls.push(message); }, appConfirm: async () => true,
    esc: value => String(value)
  });
  vm.runInContext(source, context);
  return {context, elements, calls};
}

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
