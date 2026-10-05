const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');
const { webcrypto } = require('node:crypto');

function harness() {
  const elements = new Map();
  const context = vm.createContext({
    setTimeout, clearTimeout, crypto: webcrypto, Uint8Array, TextEncoder,
    window: { crypto: webcrypto }, WebSocket: { OPEN: 1 },
    btoa: value => Buffer.from(value, 'binary').toString('base64'),
    remoteWS: null, remoteProtectionState: { blocked: false, privacy: false }, remoteFileBusy: false,
    showToast() {}, appConfirm: async () => true, appAlert: async () => {}, fmtBytes: String,
    document: { getElementById(id) {
      if (!elements.has(id)) elements.set(id, { style: {}, setAttribute() {}, textContent: '' });
      return elements.get(id);
    } }
  });
  const source = fs.readFileSync(path.join(__dirname, 'static/app.js'), 'utf8');
  vm.runInContext(source.slice(source.indexOf('function bindRemoteKeyboard('), source.indexOf('function setRemoteControls(')), context);
  vm.runInContext(source.slice(source.indexOf('function updateRemoteDeviceList()'), source.indexOf('// ==================== UTILS')), context);
  return { context, elements };
}

test('remote selection stays locked and unchanged during an active session', () => {
  const { context, elements } = harness();
  const select = context.document.getElementById('remoteDeviceSelect');
  select.value = 'original-device';
  select.innerHTML = '<option value="original-device">Original device</option>';
  for (const status of ['connecting', 'connected']) {
    context.setRemoteStatus(status);
    assert.equal(select.disabled, true);
    context.devices = [{ id: 'other-device', hostname: 'Other device', online: true }];
    context.updateRemoteDeviceList();
    assert.equal(select.value, 'original-device');
    assert.equal(select.innerHTML, '<option value="original-device">Original device</option>');
  }
  context.setRemoteStatus('disconnected');
  assert.equal(select.disabled, false);
});

test('disconnect reasons distinguish evidence from unknown network causes', () => {
  const { context } = harness();
  assert.match(context.remoteDisconnectReason({ code: 1006 }, false), /operator offline/);
  assert.match(context.remoteDisconnectReason({ reason: 'agent_connection_closed' }, true), /agent ke relay/);
  assert.match(context.remoteDisconnectReason({ reason: 'agent_connect_timeout' }, true), /30 detik/);
  assert.match(context.remoteDisconnectReason({ code: 1006 }, true), /Belum dapat membedakan/);
});

class Socket {
  readyState = 1;
  events = new Map();
  sent = [];
  offset = 0;
  addEventListener(name, callback) {
    if (!this.events.has(name)) this.events.set(name, new Set());
    this.events.get(name).add(callback);
  }
  removeEventListener(name, callback) { this.events.get(name)?.delete(callback); }
  emit(name, data) { for (const callback of [...(this.events.get(name) || [])]) callback({ data }); }
  send(payload) {
    const command = JSON.parse(payload);
    this.sent.push(command);
    if (command.type === 'file_start') this.offset = 0;
    if (command.type === 'file_chunk') {
      assert.equal(command.offset, this.offset);
      this.offset += Buffer.from(command.data, 'base64').length;
    }
    const response = command.type === 'file_end' ? { type: 'file_complete', path: 'received/report.txt' } : { type: 'file_ack', offset: this.offset };
    queueMicrotask(() => this.emit('message', JSON.stringify(response)));
  }
  close() { this.readyState = 3; this.emit('close'); }
}

test('file upload chunks, checksum and final acknowledgement', async () => {
  const { context, elements } = harness();
  const socket = new Socket();
  context.remoteWS = socket;
  const bytes = Buffer.alloc(70000, 42);
  await context.sendRemoteFile({ value: 'selected', files: [{ name: 'report.txt', size: bytes.length, arrayBuffer: async () => bytes }] });
  assert.deepEqual(socket.sent.map(command => command.type), ['file_start', 'file_chunk', 'file_chunk', 'file_end']);
  assert.equal(socket.offset, bytes.length);
  const digest = await webcrypto.subtle.digest('SHA-256', bytes);
  assert.equal(socket.sent.at(-1).digest, Buffer.from(digest).toString('hex'));
  assert.ok(elements.get('remoteFileProgress').textContent.includes('received/report.txt'));
  assert.equal(context.remoteFileBusy, false);
  assert.equal(socket.events.get('message').size, 0);
});

test('upload rejects oversize and cancelled confirmation without sending', async () => {
  const { context } = harness();
  const socket = new Socket();
  context.remoteWS = socket;
  await context.sendRemoteFile({ files: [{ name: 'huge', size: 100 * 1024 * 1024 + 1 }] });
  context.appConfirm = async () => false;
  await context.sendRemoteFile({ files: [{ name: 'cancelled', size: 1 }] });
  assert.equal(socket.sent.length, 0);
  assert.equal(context.remoteFileBusy, false);
});

test('transfer disconnect rejects pending request and removes listeners', async () => {
  const { context } = harness();
  const socket = new Socket();
  socket.send = () => {};
  const pending = context.remoteFileRequest(socket, { type: 'file_start' });
  socket.close();
  await assert.rejects(pending, /terputus/);
  assert.equal(socket.events.get('message').size, 0);
  assert.equal(socket.events.get('close').size, 0);
});

test('protection toggle waits for confirmation and never claims success locally', async () => {
  const { context } = harness();
  const socket = new Socket();
  socket.send = payload => socket.sent.push(JSON.parse(payload));
  context.remoteWS = socket;
  context.appConfirm = async () => false;
  await context.toggleRemoteProtection('block_input');
  assert.equal(socket.sent.length, 0);
  context.appConfirm = async () => true;
  await context.toggleRemoteProtection('block_input');
  assert.deepEqual(socket.sent, [{ type: 'block_input', enabled: true }]);
  assert.equal(context.remoteProtectionState.blocked, false);
});
test('remote keyboard intercepts supported paste and sends text directly', () => {
  const sent = [];
  let defaultPrevented = 0;
  const listeners = {};
  const canvas = {
    addEventListener(event, callback) { listeners[event] = callback; }
  };
  const { context } = harness();
  context.bindRemoteKeyboard(canvas, payload => sent.push(payload), () => true);
  listeners.keydown({ code: 'KeyV', ctrlKey: true, altKey: false, shiftKey: false, metaKey: false, preventDefault() { defaultPrevented++; } });
  assert.equal(sent.length, 0);
  assert.equal(defaultPrevented, 0);
  listeners.paste({
    clipboardData: { types: ['text/plain'], getData: () => 'halo remote' },
    preventDefault() { defaultPrevented++; }
  });
  assert.equal(defaultPrevented, 1);
  assert.equal(JSON.stringify(sent), JSON.stringify([{ type: 'clipboard_paste', text: 'halo remote' }]));
});

test('remote keyboard rejects paste unsupported by older agents', () => {
  const sent = [];
  const toasts = [];
  const listeners = {};
  const canvas = { addEventListener(event, callback) { listeners[event] = callback; } };
  const { context } = harness();
  context.showToast = msg => toasts.push(msg);
  context.bindRemoteKeyboard(canvas, payload => sent.push(payload), () => false);
  listeners.keydown({ code: 'KeyV', ctrlKey: true, altKey: false, shiftKey: false, metaKey: false, preventDefault() {} });
  assert.equal(JSON.stringify(sent), JSON.stringify([{ type: 'key_down', code: 'KeyV' }]));
  listeners.paste({
    clipboardData: { types: ['text/plain'], getData: () => 'halo remote' },
    preventDefault() {}
  });
  assert.match(toasts.pop() || '', /Kirim Clipboard/);
});

test('remote shortcut presets follow target OS rather than viewer OS', () => {
  const { context } = harness();
  const keys = (platform, shortcut) => Array.from(context.remoteShortcutKeys(platform, shortcut) || []);
  assert.deepEqual(keys('darwin', 'alt-tab'), ['MetaLeft', 'Tab']);
  assert.deepEqual(keys('darwin', 'task-manager'), ['MetaLeft', 'AltLeft', 'Escape']);
  assert.deepEqual(keys('darwin', 'win-d'), ['ControlLeft', 'ArrowUp']);
  assert.deepEqual(keys('windows', 'alt-tab'), ['AltLeft', 'Tab']);
  assert.deepEqual(keys(undefined, 'win-d'), ['MetaLeft', 'KeyD']);
  assert.deepEqual(keys('darwin', 'unknown'), []);
});
