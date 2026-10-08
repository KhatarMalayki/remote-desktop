const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const test = require('node:test');
const source = fs.readFileSync(require('node:path').join(__dirname, 'static/app.js'), 'utf8');
const idleSource = source.slice(source.indexOf('function resetIdleTimer()'), source.indexOf('async function saveDeviceMeta()'));
const timeout = 30 * 60 * 1000;

function tabs() {
  let now = 10000000;
  const storage = new Map();
  function tab() {
    let callback;
    const context = vm.createContext({
      token: 'session', idleTimer: null, lastIdleActivity: 0, lastIdlePublish: 0,
      IDLE_TIMEOUT_MS: timeout, Date: { now: () => now },
      localStorage: { getItem: key => storage.get(key) ?? null, setItem: (key, value) => storage.set(key, String(value)) },
      setTimeout: fn => { callback = fn; return 1; }, clearTimeout() {},
      document: { addEventListener() {} }, appAlert() {},
      doLogout() { context.loggedOut = true; }, loggedOut: false
    });
    vm.runInContext(idleSource, context);
    return { context, expire: () => callback() };
  }
  return { tab, storage, advance: milliseconds => { now += milliseconds; } };
}

test('remote activity prevents dashboard logout; true inactivity still expires', () => {
  const clock = tabs();
  const dashboard = clock.tab();
  const remote = clock.tab();
  dashboard.context.resetIdleTimer();
  remote.context.resetIdleTimer();
  clock.advance(timeout - 1000);
  remote.context.resetIdleTimer();
  clock.advance(1000);
  dashboard.expire();
  assert.equal(dashboard.context.loggedOut, false);
  clock.advance(timeout);
  dashboard.expire();
  remote.expire();
  assert.equal(dashboard.context.loggedOut, true);
  assert.equal(remote.context.loggedOut, true);
});

test('invalid or future shared activity cannot disable expiry', () => {
  for (const value of ['invalid', 'Infinity', '999999999999999']) {
    const clock = tabs();
    const browser = clock.tab();
    browser.context.resetIdleTimer();
    clock.storage.set('rd_last_activity', value);
    clock.advance(timeout);
    browser.expire();
    assert.equal(browser.context.loggedOut, true);
  }
});

test('blocked storage retains local idle protection', () => {
  const clock = tabs();
  const browser = clock.tab();
  browser.context.localStorage = { getItem() { throw Error('blocked'); }, setItem() { throw Error('blocked'); } };
  browser.context.resetIdleTimer();
  clock.advance(timeout);
  browser.expire();
  assert.equal(browser.context.loggedOut, true);
});
