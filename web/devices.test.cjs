const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');
const source = fs.readFileSync(path.join(__dirname, 'static/app.js'), 'utf8');

function harness(device) {
  const select = { value: '', innerHTML: '' };
  const calls = [];
  const context = vm.createContext({
    URLSearchParams, devices: [], remoteTabDevice: null, pendingAgentUpdates: {},
    serverAgentVersion: '0.2.58', currentUser: { role: 'admin' }, token: 'private-token',
    location: { hash: '', reload() { calls.push('reload'); } },
    window: { addEventListener(name, callback) { this[name] = callback; } },
    document: {
      title: '', body: { classList: { add(name) { calls.push(name); } } },
      getElementById() { return select; },
      createElement() { return { textContent: '', get innerHTML() {
        return String(this.textContent).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
      } }; }
    },
    api: async url => { calls.push(url); return device; },
    showPage(page) { calls.push(page); context.updateRemoteDeviceList(); },
    startRemote() { calls.push('connect:' + select.value); },
    showToast(message) { calls.push(message); },
    endpointBadges: () => 'ME / SE', osIcon: () => '', timeAgo: () => '1 menit lalu',
    isVersionNewer: () => false,
    keyStatusLabel: status => status === 'current' ? 'Key terbaru terverifikasi' : status === 'old' ? 'Key lama' : 'Key belum terverifikasi'
  });
  vm.runInContext(source.slice(source.indexOf('function buildDeviceTable('), source.indexOf('function parseAgentVersion(')), context);
  vm.runInContext(source.slice(source.indexOf('function updateRemoteDeviceList('), source.indexOf('function openSelectedRustDesk(')), context);
  vm.runInContext(source.slice(source.indexOf('function esc('), source.indexOf('function fmtBytes(')), context);
  return { context, select, calls };
}

test('device table keeps all data in six columns with safe new-tab links', () => {
  const { context } = harness();
  const device = { id: 'pc" onfocus="bad', hostname: '<Office>', assigned_to: 'Nama "Pemakai"', online: true,
    branch: 'Cabang Selatan', local_ip: '192.0.2.10', ip: '198.51.100.10', cpu_cores: 8,
    memory_used: 8, memory_total: 16, disk_used: 90, disk_total: 100, os: 'windows', arch: 'amd64', version: '0.2.58' };
  const html = context.buildDeviceTable([device]);
  assert.equal((html.match(/<th scope=/g) || []).length, 6);
  for (const value of ['&lt;Office&gt;', 'Nama &quot;Pemakai&quot;', 'Cabang Selatan', '192.0.2.10', '198.51.100.10', '8 cores', '50%', '90%', 'windows', 'amd64', 'v0.2.58', '1 menit lalu', 'ME / SE']) assert.ok(html.includes(value), value);
  assert.ok(html.includes('href="#remote=' + encodeURIComponent(device.id) + '"'));
  assert.match(html, /target="_blank" rel="noopener noreferrer"/);
  assert.doesNotMatch(html, /private-token| onfocus="bad|quickRemote/);
  assert.doesNotMatch(context.buildDeviceTable([{ ...device, online: false }]), /href="#remote=/);
});

test('resource meters clamp usage and distinguish missing data', () => {
  const { context } = harness();
  assert.match(context.deviceResourceUsage('RAM', 200, 100), /value="100"/);
  for (const values of [[0, 0], [NaN, 10], [-1, 10], [undefined, 10]]) {
    assert.match(context.deviceResourceUsage('RAM', ...values), /Belum tersedia/);
  }
});

test('remote deep link resolves a device outside the first page in its own context', async () => {
  for (const id of ['workstation-51', 'workstation-100']) {
    const { context, select, calls } = harness({ id, hostname: id, online: true });
    context.location.hash = '#remote=' + encodeURIComponent(id);
    await context.openRemoteFromLocation();
    assert.ok(calls.includes('/api/devices/' + id));
    assert.ok(calls.includes('connect:' + id));
    assert.equal(context.document.title, id + ' — RemoteDesk');
    assert.equal(context.devices.length, 0);
    context.updateRemoteDeviceList();
    assert.equal(select.value, id);
    assert.ok(select.innerHTML.includes(id));
  }
});

test('remote deep links never connect offline, denied, or invalid targets', async () => {
  for (const device of [null, { error: 'forbidden' }, { id: 'other', online: true }, { id: 'target', online: false }]) {
    const { context, calls } = harness(device);
    context.location.hash = '#remote=target';
    await context.openRemoteFromLocation();
    assert.ok(!calls.some(value => value.startsWith('connect:')));
  }
  const { context, calls } = harness();
  await context.openRemoteFromLocation();
  assert.deepEqual(calls, []);
  context.location.hash = '#remote=%00';
  await context.openRemoteFromLocation();
  assert.ok(!calls.some(value => value.startsWith('/api/')));
});

test('authentication changes in another tab reload the current session', () => {
  const { context, calls } = harness();
  vm.runInContext(source.slice(source.indexOf("window.addEventListener('storage'"), source.indexOf('loadServerVersion();', source.indexOf('// ==================== BOOT'))), context);
  context.window.storage({ key: 'other', newValue: null });
  context.window.storage({ key: 'rd_token', newValue: 'private-token' });
  assert.deepEqual(calls, []);
  context.window.storage({ key: 'rd_token', newValue: null });
  assert.deepEqual(calls, ['reload']);
});
