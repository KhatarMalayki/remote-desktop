const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');

test('browser popup calls are absent from application scripts', () => {
  for (const name of ['app.js', 'ownership.js']) {
    const source = fs.readFileSync(path.join(__dirname, 'static', name), 'utf8');
    assert.doesNotMatch(source, /\b(?:alert|confirm|prompt)\s*\(/);
    new vm.Script(source);
  }
});

test('dialog confirmation, cancellation, queue, and secret cleanup', async () => {
  let active;
  let restored = 0;
  const previous = { isConnected: true, focus() { restored++; } };
  function element() {
    return { events: {}, addEventListener(name, callback) { this.events[name] = callback; }, focus() {}, select() {} };
  }
  const document = {
    activeElement: previous,
    body: { appendChild(dialog) { active = dialog; } },
    createElement() {
      const dialog = element();
      const children = new Map();
      Object.assign(dialog, {
        setAttribute() {},
        querySelector(selector) {
          if (!children.has(selector)) children.set(selector, element());
          return children.get(selector);
        },
        showModal() {},
        close() { this.events.close(); },
        remove() { active = null; }
      });
      return dialog;
    }
  };
  const context = vm.createContext({ document });
  vm.runInContext(fs.readFileSync(path.join(__dirname, 'static/dialogs.js'), 'utf8'), context);
  const confirmation = context.appConfirm('<img src=x onerror=bad()>');
  const prompt = context.appPrompt('Password', '', { type: 'password' });
  await Promise.resolve();
  assert.equal(active.querySelector('p').textContent, '<img src=x onerror=bad()>');
  active.querySelector('[data-cancel]').events.click();
  assert.equal(await confirmation, false);
  await Promise.resolve();
  const input = active.querySelector('input');
  assert.equal(input.type, 'password');
  input.value = 'secret';
  active.querySelector('form').events.submit({ preventDefault() {} });
  assert.equal(await prompt, 'secret');
  assert.equal(input.value, '');
  const cancelled = context.appPrompt('Name');
  await Promise.resolve();
  active.close();
  assert.equal(await cancelled, null);
  const accepted = context.appConfirm('Proceed?');
  await Promise.resolve();
  active.querySelector('form').events.submit({ preventDefault() {} });
  assert.equal(await accepted, true);
  assert.equal(restored, 4);
});
