const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');

const appSource = fs.readFileSync(path.join(__dirname, 'static/app.js'), 'utf8');
const indexHtml = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');

test('index.html contains accessible structure for usersModal and editUserModal', () => {
  assert.ok(indexHtml.includes('class="modal users-modal"'));
  assert.ok(indexHtml.includes('class="modal users-modal-edit"'));
  assert.ok(indexHtml.includes('id="usersModalTitle"'));
  assert.ok(indexHtml.includes('id="editUserModalTitle"'));
  assert.ok(indexHtml.includes('for="newUserUsername"'));
  assert.ok(indexHtml.includes('for="newUserPassword"'));
  assert.ok(indexHtml.includes('for="newUserRole"'));
  assert.ok(indexHtml.includes('for="newUserBranch"'));
  assert.ok(indexHtml.includes('for="editUserUsername"'));
  assert.ok(indexHtml.includes('for="editUserPassword"'));
  assert.ok(indexHtml.includes('for="editUserRole"'));
  assert.ok(indexHtml.includes('for="editUserBranch"'));
  assert.ok(indexHtml.includes('class="users-select-multiple"'));
  assert.ok(!indexHtml.includes('ðŸ’¡'));
  assert.ok(!indexHtml.includes('ðŸ‘‘'));
});

test('loadUsers renders structured table with user metadata, badges, and action group', async () => {
  const userSection = appSource.slice(
    appSource.indexOf('// ==================== USER MANAGEMENT ===================='),
    appSource.indexOf('// ==================== AUDIT HISTORY ====================')
  );

  const container = { innerHTML: '' };
  const countBadge = { textContent: '' };
  const elements = {
    usersTableContainer: container,
    usersCountBadge: countBadge,
    usersModal: { style: { display: 'none' } }
  };

  const usersMock = [
    { id: 1, username: 'admin', role: 'admin', branch: 'Pusat', mfa_enabled: true },
    { id: 2, username: 'adh_sby', role: 'adh', branch: 'Surabaya, Malang', mfa_enabled: false }
  ];

  const context = vm.createContext({
    currentUser: { username: 'admin', role: 'admin' },
    document: {
      getElementById: (id) => elements[id] || null
    },
    api: async (url) => {
      if (url === '/api/users') return usersMock;
      return [];
    },
    esc: (str) => String(str || '')
  });

  vm.runInContext(userSection, context);
  await context.loadUsers();

  assert.equal(countBadge.textContent, '2 akun');
  assert.ok(container.innerHTML.includes('<table class="device-table">'));
  assert.ok(container.innerHTML.includes('<th>Username</th><th>Role</th><th>Cabang</th><th>Aksi</th>'));
  assert.ok(container.innerHTML.includes('user-cell-meta'));
  assert.ok(container.innerHTML.includes('user-self-pill'));
  assert.ok(container.innerHTML.includes('2FA ON'));
  assert.ok(container.innerHTML.includes('user-badge adh'));
  assert.ok(container.innerHTML.includes('user-action-group'));
  assert.ok(container.innerHTML.includes('openEditUserModal(2)'));
});

test('loadUsers handles empty user list cleanly', async () => {
  const userSection = appSource.slice(
    appSource.indexOf('// ==================== USER MANAGEMENT ===================='),
    appSource.indexOf('// ==================== AUDIT HISTORY ====================')
  );

  const container = { innerHTML: '' };
  const countBadge = { textContent: '' };
  const elements = {
    usersTableContainer: container,
    usersCountBadge: countBadge
  };

  const context = vm.createContext({
    currentUser: { username: 'admin' },
    document: {
      getElementById: (id) => elements[id] || null
    },
    api: async () => [],
    esc: (s) => String(s || '')
  });

  vm.runInContext(userSection, context);
  await context.loadUsers();

  assert.ok(container.innerHTML.includes('Belum ada user terdaftar'));
});
