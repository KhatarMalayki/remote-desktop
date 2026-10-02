const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');

test('Mac links and ZIP downloads retain the selected architecture', async () => {
  const source = fs.readFileSync(path.join(__dirname, 'static/app.js'), 'utf8');
  for (const arch of ['arm64', 'amd64']) {
    let copied;
    const anchor = { setAttribute() {}, click() {} };
    const context = vm.createContext({
      currentUser: { role: 'admin' }, token: 'fixture-token',
      window: { location: { origin: 'https://fixture.invalid' } },
      navigator: { clipboard: { writeText: async value => { copied = value; } } },
      showToast() {}, appAlert(message) { throw new Error(message); }, setTimeout() {},
      document: {
        getElementById(id) {
          return id === 'dlAgentOS' ? { value: 'darwin', selectedOptions: [{ dataset: { arch } }] } : { value: 'Mac Office' };
        },
        createElement() { return anchor; },
        body: { appendChild() {}, removeChild() {} }
      }
    });
    vm.runInContext(source.slice(source.indexOf('function copyDownloadAgentLink('), source.indexOf('// ==================== MASTER ROLE')), context);
    context.copyDownloadAgentLink();
    context.submitDownloadAgentPackage();
    for (const value of [copied, anchor.href]) {
      const url = new URL(value, 'https://fixture.invalid');
      assert.equal(url.searchParams.get('os'), 'darwin');
      assert.equal(url.searchParams.get('arch'), arch);
      assert.equal(url.searchParams.get('branch'), 'Mac Office');
    }
  }
});
