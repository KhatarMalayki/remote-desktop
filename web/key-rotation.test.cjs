const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');
const source = fs.readFileSync(path.join(__dirname, 'static/app.js'), 'utf8');
const rotation = source.slice(source.indexOf('async function openReconfigureModal()'), source.indexOf('// ==================== CHANGE & RESET PASSWORD'));

function harness() {
  const fields = {
    rotationKey: { value: 'special-#&+%?= key' },
    rotationDevice: { value: 'pilot' },
    rotationEndpoint: { value: 'https://example.test' },
    keyRotationResult: { textContent: '', scrollIntoView() { this.scrolled = true; } }
  };
  const requests = [];
  const buttons = [{ disabled: false }];
  const context = vm.createContext({
    document: { getElementById: id => fields[id], querySelectorAll: () => buttons },
    appConfirm: async () => true,
    api: async (url, options) => { requests.push(JSON.parse(options.body)); return { status: 'ok' }; },
    loadDevices: async () => {}
  });
  vm.runInContext(rotation, context);
  context.openReconfigureModal = async () => {};
  return { context, fields, requests, buttons };
}

test('prepare preserves special characters without selecting or broadcasting to devices', async () => {
  const { context, fields, requests, buttons } = harness();
  await context.submitKeyRotation('prepare');
  assert.deepEqual(requests, [{ operation: 'prepare', api_key: 'special-#&+%?= key' }]);
  assert.equal(fields.rotationKey.value, '');
  assert.equal(buttons[0].disabled, false);
});

test('migration targets one device without resending secret or claiming success', async () => {
  const { context, fields, requests } = harness();
  await context.submitKeyRotation('migrate');
  assert.deepEqual(requests, [{ operation: 'migrate', device_id: 'pilot' }]);
  assert.match(fields.keyRotationResult.textContent, /Belum dinyatakan berhasil/);
  assert.equal(context.keyStatusLabel('unknown'), 'Key belum terverifikasi');
  assert.equal(context.keyStatusLabel('current'), 'Key terbaru terverifikasi');
});

test('cancel sends nothing and failed request does not claim success', async () => {
  const { context, fields, requests, buttons } = harness();
  context.appConfirm = async () => false;
  await context.submitKeyRotation('revoke');
  assert.equal(requests.length, 0);
  context.appConfirm = async () => true;
  context.api = async () => ({ error: 'Device offline' });
  await context.submitKeyRotation('migrate');
  assert.equal(fields.keyRotationResult.textContent, 'Device offline');
  assert.equal(fields.keyRotationResult.scrolled, true);
  assert.equal(buttons[0].disabled, false);
});

test('legacy recovery uses retain without selecting devices or changing the key', async () => {
  const { context, fields, requests } = harness();
  fields.rotationKey.value = 'legacy#1234';
  await context.submitKeyRotation('retain');
  assert.deepEqual(requests, [{ operation: 'retain', api_key: 'legacy#1234' }]);
});

test('bulk migration sends only an explicit operation and reports partial delivery', async () => {
  const { context, fields, requests } = harness();
  context.api = async (url, options) => {
    requests.push(JSON.parse(options.body));
    return { counts: { sent: 3, current: 1, offline: 2, update_required: 4, failed: 1 } };
  };
  await context.submitKeyRotation('migrate_all');
  assert.deepEqual(requests, [{ operation: 'migrate_all' }]);
  assert.match(fields.keyRotationResult.textContent, /Perintah terkirim: 3/);
  assert.match(fields.keyRotationResult.textContent, /Offline: 2/);
  assert.match(fields.keyRotationResult.textContent, /Perlu update agent: 4/);
  assert.match(fields.keyRotationResult.textContent, /Belum berarti migrasi berhasil/);
});

test('rotation result is located directly below the secret input', () => {
  assert.ok(rotation.indexOf('id="keyRotationResult"') > rotation.indexOf('id="rotationKey"'));
  assert.ok(rotation.indexOf('id="keyRotationResult"') < rotation.indexOf('data-key-operation="prepare"'));
});
