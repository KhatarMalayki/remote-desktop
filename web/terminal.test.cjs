const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');
test('failed commands retain diagnostic output', async () => {
  const source = fs.readFileSync(path.join(__dirname, 'static/app.js'), 'utf8');
  const command = source.slice(source.indexOf('async function runTerminalCommand()'), source.indexOf('// ==================== UTILS'));
  const elements = { terminalInput: { value: 'net user', focus() {} }, terminalShellSelect: { value: 'cmd' }, terminalOutput: { textContent: '' }, btnRunCommand: {} };
  const context = vm.createContext({ currentTerminalDeviceID: 'test', document: { getElementById: id => elements[id] }, api: async () => ({ stdout: 'partial output', stderr: 'actual failure', error: 'exit status 1', exit_code: 1 }) });
  vm.runInContext(command, context);
  await context.runTerminalCommand();
  for (const expected of ['partial output', 'actual failure', 'exit status 1', 'Exit code: 1']) assert.ok(elements.terminalOutput.textContent.includes(expected));
  assert.equal(elements.btnRunCommand.disabled, false);
});
