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
    keyRotationResult: { textContent: '' }
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
  assert.equal(buttons[0].disabled, false);
});
