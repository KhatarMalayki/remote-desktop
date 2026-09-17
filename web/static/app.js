// RemoteDesk - Remote Desktop & Branch Asset Verification
// Frontend Application

let token = localStorage.getItem('rd_token') || '';
let currentUser = null;
let currentPage = 'dashboard';
let currentAssetTab = 'all';
let devices = [];
let manualAssets = [];
let currentDevice = null;
let ws = null;
let remoteWS = null;
let searchTimeout = null;
let idleTimer = null;
const IDLE_TIMEOUT_MS = 30 * 60 * 1000;
let currentOffset = 0;
const PAGE_SIZE = 50;

var chartOnlineOffline = null;
var chartOS = null;
var chartMemory = null;
var chartDisk = null;

const CHART_COLORS = [
  '#4f8cff', '#34d399', '#fbbf24', '#ef4444', '#a78bfa',
  '#f472b6', '#38bdf8', '#fb923c', '#94a3b8', '#6ee7b7'
];

// ==================== AUTH ====================

async function loadServerVersion() {
  try {
    const res = await fetch('/api/agent/version', { cache: 'no-store' });
    const data = await res.json();
    const label = 'Server v' + (data.version || 'tidak diketahui');
    const sidebar = document.getElementById('sidebarVersion');
    const login = document.getElementById('loginServerVersion');
    if (sidebar) sidebar.textContent = label + ' • Enterprise Asset';
    if (login) login.textContent = label;
  } catch (e) {
    const login = document.getElementById('loginServerVersion');
    if (login) login.textContent = 'Versi server tidak dapat dibaca';
  }
}

async function checkAuth() {
  if (!token) {
    document.getElementById('loginPage').style.display = 'flex';
    document.getElementById('appContainer').style.display = 'none';
    return;
  }
  const me = await api('/api/auth/me');
  if (me && me.username) {
    currentUser = me;
    updateUserUI();
    document.getElementById('loginPage').style.display = 'none';
    document.getElementById('appContainer').style.display = 'flex';
    init();
  } else {
    doLogout();
  }
}

async function doLogin() {
  const user = document.getElementById('loginUser').value;
  const pass = document.getElementById('loginPass').value;
  try {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: user, password: pass, remember_device: !!(document.getElementById('rememberDevice') || {}).checked })
    });
    const data = await res.json();
    if (res.ok && data.mfa_required) {
      document.getElementById('loginMFATicket').value = data.mfa_ticket || '';
      document.getElementById('loginCreds').style.display = 'none';
      document.getElementById('loginMFA').style.display = 'block';
      const codeInput = document.getElementById('loginMFACode');
      if (codeInput) { codeInput.value = ''; codeInput.focus(); }
      const errEl = document.getElementById('loginError');
      if (errEl) errEl.style.display = 'none';
      return;
    }
    if (res.ok && data.token) {
      token = data.token;
      localStorage.setItem('rd_token', token);
      currentUser = { username: data.username, role: data.role, branch: data.branch };
      updateUserUI();
      document.getElementById('loginPage').style.display = 'none';
      document.getElementById('appContainer').style.display = 'flex';
      init();
    } else {
      const errEl = document.getElementById('loginError');
      errEl.textContent = data.error || 'Login failed';
      errEl.style.display = 'block';
    }
  } catch (e) {
    const errEl = document.getElementById('loginError');
    errEl.textContent = 'Connection error';
    errEl.style.display = 'block';
  }
}

function doLogout() {
  token = '';
  currentUser = null;
  localStorage.removeItem('rd_token');
  location.reload();
}

function updateUserUI() {
  if (!currentUser) return;
  const nameEl = document.getElementById('sidebarUsername');
  const badgeEl = document.getElementById('sidebarBadge');
  const navUsers = document.getElementById('navUsers');
  if (nameEl) nameEl.textContent = currentUser.username;
  if (badgeEl) {
    if (currentUser.role === 'admin') {
      badgeEl.textContent = 'Administrator';
      badgeEl.className = 'user-badge admin';
    } else if (currentUser.role === 'adh') {
      badgeEl.textContent = 'ADH: ' + (currentUser.branch || 'Cabang');
      badgeEl.className = 'user-badge adh';
    } else {
      badgeEl.textContent = currentUser.role.toUpperCase();
      badgeEl.className = 'user-badge viewer';
    }
  }
  if (navUsers) navUsers.style.display = currentUser.role === 'admin' ? 'flex' : 'none';
  var navBranches = document.getElementById('navBranches');
  if (navBranches) navBranches.style.display = currentUser.role === 'admin' ? 'flex' : 'none';
  var navReconf = document.getElementById('navReconfigure');
  if (navReconf) navReconf.style.display = currentUser.role === 'admin' ? 'flex' : 'none';
  var navSec = document.getElementById('navSecurityLogs');
  if (navSec) navSec.style.display = currentUser.role === 'admin' ? 'flex' : 'none';
  var adminHdr = document.getElementById('adminSectionHeader');
  if (adminHdr) adminHdr.style.display = currentUser.role === 'admin' ? 'block' : 'none';
  var bTitle = document.getElementById('branchBannerTitle');
  var bSub = document.getElementById('branchBannerSubtitle');
  if (currentUser.role === 'adh') {
    if (bTitle) bTitle.textContent = 'Inventaris & Verifikasi Cabang ' + (currentUser.branch || '');
    if (bSub) bSub.textContent = 'Portal verifikasi fisik dan pendaftaran aset operasional cabang ' + (currentUser.branch || '') + '.';
  }
}

// ==================== INIT ====================

function init() {
  ownershipUI();
  loadStats();
  loadDevices();
  loadGroups();
  loadBranches();
  loadBranchAssets();
  if (!isAssetUser()) connectWS();
  setInterval(loadStats, 30000);
  setInterval(loadDevices, 30000);
  setInterval(loadBranchAssets, 30000);
}

// ==================== API ====================

async function api(path, opts) {
  opts = opts || {};
  try {
    var res = await fetch(path, Object.assign({}, opts, {
      headers: Object.assign({ 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' }, opts.headers || {})
    }));
    if (res.status === 401) { doLogout(); return null; }
    return await res.json();
  } catch (e) { console.error('API Error:', e); return null; }
}
// ==================== WEBSOCKET ====================

function connectWS() {
  var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  ws = new WebSocket(proto + '//' + location.host + '/ws/viewer?token=' + token);
  ws.onopen = function() { console.log('[ws] connected'); };
  ws.onmessage = function(e) {
    try {
      var msg = JSON.parse(e.data);
      if (msg.action === 'device_online') updateOnlineStatus(msg.data.id, true);
      else if (msg.action === 'device_offline') updateOnlineStatus(msg.data.id, false);
      else if (msg.action === 'signal') handleSignal(msg.data);
    } catch(ex) {}
  };
  ws.onclose = function() { setTimeout(connectWS, 3000); };
}

function updateOnlineStatus(id, online) {
  var dev = devices.find(function(d) { return d.id === id; });
  if (dev) dev.online = online;
  if (currentPage === 'devices') renderDevices();
  if (currentPage === 'dashboard') { renderRecentDevices(); renderCharts(); }
  if (currentPage === 'branch-assets') renderBranchAssets();
  updateRemoteDeviceList();
}

// ==================== PAGES ====================

function showPage(page) {
  currentPage = page;
  document.querySelectorAll('.page').forEach(function(p) { p.classList.remove('active'); });
  var el = document.getElementById('page-' + page);
  if (el) el.classList.add('active');
  document.querySelectorAll('.nav-item[data-page]').forEach(function(n) {
    n.classList.toggle('active', n.dataset.page === page);
  });
  if (page === 'dashboard') { loadStats(); loadDevices(); }
  if (page === 'devices') { loadDevices(); }
  if (page === 'assets') { loadDevices(); }
  if (page === 'branch-assets') { loadBranchAssets(); }
  if (page === 'remote') { updateRemoteDeviceList(); }
}

var lastStats = null;

async function loadStats() {
  var data = await api('/api/stats');
  if (!data) return;
  lastStats = data;
  document.getElementById('statTotal').textContent = data.total_devices || 0;
  document.getElementById('statOnline').textContent = data.online_devices || 0;
  document.getElementById('statOffline').textContent = data.offline_devices || 0;
  if (currentPage === 'dashboard') renderCharts();
}

function renderRecentDevices() {
  var recent = devices.slice(0, 10);
  if (recent.length === 0) {
    document.getElementById('recentDevices').innerHTML = '<div class="empty-state"><p>Belum ada perangkat.</p></div>';
    return;
  }
  document.getElementById('recentDevices').innerHTML = buildDeviceTable(recent);
}

function renderCharts() { if (!lastStats) return; renderOnlineOfflineChart(); renderOSChart(); renderMemoryChart(); renderDiskChart(); }

function chartDefaults() {
  return { responsive: true, maintainAspectRatio: false, plugins: { legend: { labels: { color: '#a0a3b1', font: { size: 12 } } } } };
}

function renderOnlineOfflineChart() {
  var ctx = document.getElementById('chartOnlineOffline'); if (!ctx) return;
  if (chartOnlineOffline) chartOnlineOffline.destroy();
  chartOnlineOffline = new Chart(ctx, { type: 'doughnut', data: { labels: ['Online','Offline'], datasets: [{ data: [lastStats.online_devices||0, lastStats.offline_devices||0], backgroundColor: ['#34d399','#ef4444'], borderWidth: 0 }] }, options: Object.assign({}, chartDefaults(), { cutout: '70%', plugins: { legend: { position: 'bottom', labels: { color: '#a0a3b1', boxWidth: 12, padding: 12 } } } }) });
}

function renderOSChart() {
  var ctx = document.getElementById('chartOS'); if (!ctx) return;
  var osData = lastStats.os_distribution || {}; var labels = Object.keys(osData); var values = Object.values(osData);
  if (chartOS) chartOS.destroy();
  chartOS = new Chart(ctx, { type: 'doughnut', data: { labels: labels.length ? labels : ['No Data'], datasets: [{ data: values.length ? values : [1], backgroundColor: CHART_COLORS.slice(0, Math.max(labels.length,1)), borderWidth: 0 }] }, options: Object.assign({}, chartDefaults(), { cutout: '70%', plugins: { legend: { position: 'bottom', labels: { color: '#a0a3b1', boxWidth: 12, padding: 12 } } } }) });
}

function renderMemoryChart() {
  var ctx = document.getElementById('chartMemory'); if (!ctx) return;
  var sorted = devices.filter(function(d){return d.memory_total>0}).sort(function(a,b){return (b.memory_used/b.memory_total)-(a.memory_used/a.memory_total)}).slice(0,10);
  var lb = sorted.map(function(d){return d.hostname||d.id}); var pc = sorted.map(function(d){return Math.round(d.memory_used/d.memory_total*100)});
  if (chartMemory) chartMemory.destroy();
  chartMemory = new Chart(ctx, { type:'bar', data:{labels:lb.length?lb:['No Data'], datasets:[{label:'RAM %',data:pc.length?pc:[0],backgroundColor:'#4f8cff',borderRadius:4}]}, options:Object.assign({},chartDefaults(),{indexAxis:'y',scales:{x:{min:0,max:100,ticks:{color:'#a0a3b1',callback:function(v){return v+'%'}},grid:{color:'#2d3042'}},y:{ticks:{color:'#a0a3b1'},grid:{display:false}}},plugins:{legend:{display:false}}}) });
}

function renderDiskChart() {
  var ctx = document.getElementById('chartDisk'); if (!ctx) return;
  var sorted = devices.filter(function(d){return d.disk_total>0}).sort(function(a,b){return (b.disk_used/b.disk_total)-(a.disk_used/a.disk_total)}).slice(0,10);
  var lb = sorted.map(function(d){return d.hostname||d.id}); var pc = sorted.map(function(d){return Math.round(d.disk_used/d.disk_total*100)});
  if (chartDisk) chartDisk.destroy();
  chartDisk = new Chart(ctx, { type:'bar', data:{labels:lb.length?lb:['No Data'], datasets:[{label:'Disk %',data:pc.length?pc:[0],backgroundColor:'#a78bfa',borderRadius:4}]}, options:Object.assign({},chartDefaults(),{indexAxis:'y',scales:{x:{min:0,max:100,ticks:{color:'#a0a3b1',callback:function(v){return v+'%'}},grid:{color:'#2d3042'}},y:{ticks:{color:'#a0a3b1'},grid:{display:false}}},plugins:{legend:{display:false}}}) });
}

// ==================== DEVICES ====================

async function loadDevices() {
  var group = (document.getElementById('groupFilter') || {}).value || '';
  var search = (document.getElementById('deviceSearch') || {}).value || '';
  var data = await api('/api/devices?group=' + encodeURIComponent(group) + '&search=' + encodeURIComponent(search) + '&offset=' + currentOffset + '&limit=' + PAGE_SIZE);
  if (!data) return;
  devices = data.devices || [];
  if (currentPage === 'devices') renderDevices();
  if (currentPage === 'dashboard') renderRecentDevices();
  if (currentPage === 'assets') renderAssets();
  renderPagination(data.total, data.offset, data.limit);
}

function renderDevices() {
  var sf = (document.getElementById('statusFilter') || {}).value || '';
  var filtered = devices;
  if (sf === 'online') filtered = devices.filter(function(d){return d.online});
  if (sf === 'offline') filtered = devices.filter(function(d){return !d.online});
  if (filtered.length === 0) { document.getElementById('devicesTable').innerHTML = '<div class="empty-state"><p>No devices found</p></div>'; return; }
  document.getElementById('devicesTable').innerHTML = buildDeviceTable(filtered);
}

function buildDeviceTable(list) {
  return '<table class="device-table"><thead><tr><th>Status</th><th>Hostname</th><th>OS</th><th>IP</th><th>CPU</th><th>RAM</th><th>Disk</th><th>Group</th><th>Last Seen</th><th>Actions</th></tr></thead><tbody>' +
    list.map(function(d) {
      var ramPct = d.memory_total ? Math.round(d.memory_used / d.memory_total * 100) : 0;
      var diskPct = d.disk_total ? Math.round(d.disk_used / d.disk_total * 100) : 0;
      return '<tr><td><span class="status-dot '+(d.online?'online':'offline')+'"></span>'+(d.online?'Online':'Offline')+'</td>' +
        '<td><strong>'+esc(d.hostname)+'</strong><br><small style="color:var(--fg2)">'+esc(d.id)+'</small></td>' +
        '<td>'+osIcon(d.os)+' '+esc(d.os)+' '+esc(d.arch)+'</td>' +
        '<td>'+esc(d.ip)+'</td>' +
        '<td>'+d.cpu_cores+' cores</td>' +
        '<td><div class="progress-bar"><div class="fill '+(ramPct>80?'danger':'')+'" style="width:'+ramPct+'%"></div></div>'+ramPct+'%</td>' +
        '<td><div class="progress-bar"><div class="fill '+(diskPct>80?'danger':'')+'" style="width:'+diskPct+'%"></div></div>'+diskPct+'%</td>' +
        '<td><span class="tag">'+esc(d.branch||d.group||'default')+'</span></td>' +
        '<td>'+timeAgo(d.last_seen)+'</td>' +
        '<td><button class="btn btn-ghost btn-sm" onclick="openDeviceModal(\''+d.id+'\')">Details</button>' +
        (d.online ? ' <button class="btn btn-primary btn-sm" onclick="quickRemote(\''+d.id+'\')">Remote</button>' : '') +
        '</td></tr>';
    }).join('') + '</tbody></table>';
}

function searchDevices() { clearTimeout(searchTimeout); searchTimeout = setTimeout(function(){ currentOffset=0; loadDevices(); }, 300); }
function filterDevices() { currentOffset=0; loadDevices(); }

function renderPagination(total, offset, limit) {
  var el = document.getElementById('pagination');
  if (!el || total <= limit) { if(el) el.innerHTML=''; return; }
  var totalPages = Math.ceil(total / limit);
  var cur = Math.floor(offset / limit) + 1;
  var html = '';
  for (var i = 1; i <= totalPages; i++) html += '<button class="page-btn '+(i===cur?'active':'')+'" onclick="goPage('+i+')">'+i+'</button>';
  el.innerHTML = html;
}

function goPage(page) { currentOffset = (page-1)*PAGE_SIZE; loadDevices(); }

async function loadGroups() {
  var groups = await api('/api/groups');
  if (!groups) return;
  var sel = document.getElementById('groupFilter');
  if (!sel) return;
  var cur = sel.value;
  sel.innerHTML = '<option value="">All Groups</option>';
  groups.forEach(function(g) { sel.innerHTML += '<option value="'+esc(g)+'">'+esc(g)+'</option>'; });
  sel.value = cur;
}

// ==================== BRANCH ASSETS & VERIFICATION ====================

async function loadBranches() {
  if (isAssetUser()) { document.getElementById('branchSelectFilter').innerHTML = '<option value="">Aset Saya</option>'; return; }
  var branches = await api('/api/branches');
  var sel = document.getElementById('branchSelectFilter');
  if (!sel) return;
  var cur = sel.value;
  sel.innerHTML = currentUser && currentUser.role === 'adh' ? '' : '<option value="">Semua Cabang</option>';
  (branches || []).forEach(function(b) { sel.innerHTML += '<option value="'+esc(b)+'">'+esc(b)+'</option>'; });
  if (currentUser && currentUser.role === 'adh' && currentUser.branch) { sel.value = currentUser.branch; sel.disabled = true; }
  else if (cur) { sel.value = cur; }
}

function onBranchChange() { loadBranchAssets(); }

function setAssetTab(tab) {
  currentAssetTab = tab;
  document.querySelectorAll('.tab-btn').forEach(function(b){b.classList.remove('active')});
  var id = 'tabBtn' + tab.charAt(0).toUpperCase() + tab.slice(1);
  var btn = document.getElementById(id);
  if (btn) btn.classList.add('active');
  renderBranchAssets();
}

async function loadBranchAssets() {
  var branch = (document.getElementById('branchSelectFilter')||{}).value || '';
  if (currentUser && currentUser.role === 'adh' && currentUser.branch) branch = currentUser.branch;
  var cat = (document.getElementById('branchCategoryFilter')||{}).value || '';
  var vs = (document.getElementById('branchStatusFilter')||{}).value || '';
  var search = (document.getElementById('branchAssetSearch')||{}).value || '';

  var results = await Promise.all([
    api('/api/assets/manual?branch='+encodeURIComponent(branch)+'&category='+encodeURIComponent(cat)+'&verification_status='+encodeURIComponent(vs)+'&search='+encodeURIComponent(search)),
    api('/api/branches/stats?branch='+encodeURIComponent(branch)),
    api('/api/devices?group='+encodeURIComponent(branch)+'&limit=200')
  ]);
  manualAssets = results[0] || [];
  var stats = results[1];
  if (results[2] && results[2].devices) devices = results[2].devices;
  if (isAssetUser()) {
    var owned = manualAssets.concat(devices);
    stats = {total_assets:owned.length, verified:0, unverified:0, discrepancy:0};
    owned.forEach(function(a) { var key = a.verification_status || 'unverified'; if (key in stats) stats[key]++; });
  }

  if (stats) {
    document.getElementById('branchStatTotal').textContent = stats.total_assets || 0;
    document.getElementById('branchStatVerified').textContent = stats.verified || 0;
    document.getElementById('branchStatUnverified').textContent = stats.unverified || 0;
    document.getElementById('branchStatDiscrepancy').textContent = stats.discrepancy || 0;
  }
  renderBranchAssets();
}

function onSearchBranchAssets() { clearTimeout(searchTimeout); searchTimeout = setTimeout(renderBranchAssets, 250); }

function renderBranchAssets() {
  var container = document.getElementById('branchAssetsTableContainer');
  if (!container) return;
  var search = ((document.getElementById('branchAssetSearch')||{}).value || '').toLowerCase();
  var catFilter = (document.getElementById('branchCategoryFilter')||{}).value || '';
  var statusFilter = (document.getElementById('branchStatusFilter')||{}).value || '';
  var branchFilter = (document.getElementById('branchSelectFilter')||{}).value || '';
  if (currentUser && currentUser.role === 'adh' && currentUser.branch) branchFilter = currentUser.branch;

  var items = [];
  if (currentAssetTab === 'all' || currentAssetTab === 'manual') {
    manualAssets.forEach(function(a) {
      items.push({ isManual:true, id:a.id, tag:a.asset_tag, name:a.name, category:a.category, branch:a.branch, location:a.location, pic:a.assigned_to, condition:a.condition, specs:a.specs||a.serial_number, vStatus:a.verification_status||'unverified', vAt:a.verified_at, vBy:a.verified_by, vNote:a.verification_note });
    });
  }
  if (currentAssetTab === 'all' || currentAssetTab === 'device') {
    devices.forEach(function(d) {
      var dBranch = d.branch || d.group || 'default';
      if (branchFilter && dBranch !== branchFilter) return;
      items.push({ isManual:false, id:d.id, tag:d.hostname, name:d.os+' '+d.arch+' ('+d.hostname+')', category:'pc', branch:dBranch, location:d.local_ip||d.ip, pic:d.note||'-', condition:d.online?'good':'fair', specs:d.cpu_model+' ('+d.cpu_cores+'c) | '+fmtBytes(d.memory_total), vStatus:d.verification_status||'unverified', vAt:d.verified_at, vBy:d.verified_by, vNote:d.verification_note });
    });
  }

  if (catFilter) items = items.filter(function(it){return it.category===catFilter});
  if (statusFilter) items = items.filter(function(it){return it.vStatus===statusFilter});
  if (search) items = items.filter(function(it){return (it.name||'').toLowerCase().includes(search)||(it.tag||'').toLowerCase().includes(search)||(it.location||'').toLowerCase().includes(search)||(it.pic||'').toLowerCase().includes(search)||(it.specs||'').toLowerCase().includes(search)});

  if (items.length === 0) {
    container.innerHTML = '<div class="empty-state"><p>Tidak ada aset ditemukan. Klik "+ Input Aset Manual" untuk menambahkan.</p></div>';
    return;
  }

  container.innerHTML = '<table class="device-table"><thead><tr><th>Tag / Hostname</th><th>Nama Perangkat</th><th>Kategori</th><th>Cabang & Lokasi</th><th>PIC</th><th>Kondisi</th><th>Status Verifikasi</th><th>Verifikator</th><th>Aksi</th></tr></thead><tbody>' +
    items.map(function(it) {
      var vBadge = it.vStatus==='verified'?'verified':(it.vStatus==='discrepancy'?'discrepancy':'unverified');
      var vLabel = it.vStatus==='verified'?'&#10004; Terverifikasi':(it.vStatus==='discrepancy'?'&#10008; Selisih':'&#9203; Belum');
      var condClass = it.condition==='good'?'good':(it.condition==='fair'?'fair':'damaged');
      var condLabel = it.condition==='good'?'Baik':(it.condition==='fair'?'Rusak Ringan':'Rusak Berat');
      return '<tr>' +
        '<td><strong>'+esc(it.tag)+'</strong><br><small style="color:var(--fg2)">'+(it.isManual?'Manual':'Agent')+'</small></td>' +
        '<td><strong>'+esc(it.name)+'</strong><br><small style="color:var(--fg2)">'+esc(it.specs||'-')+'</small></td>' +
        '<td><span class="tag">'+esc(it.category.toUpperCase())+'</span></td>' +
        '<td>'+esc(it.branch)+'<br><small style="color:var(--fg2)">'+esc(it.location||'-')+'</small></td>' +
        '<td>'+esc(it.pic||'-')+'</td>' +
        '<td><span class="badge-cond '+condClass+'">'+condLabel+'</span></td>' +
        '<td><span class="badge-status '+vBadge+'">'+vLabel+'</span></td>' +
        '<td>'+(it.vBy?'<strong>'+esc(it.vBy)+'</strong><br><small style="color:var(--fg2)">'+(it.vAt?new Date(it.vAt).toLocaleDateString():'')+' '+esc(it.vNote||'')+'</small>':'<span style="color:var(--fg2);font-size:11px">Belum dicek</span>')+'</td>' +
        '<td><div style="display:flex;gap:4px;flex-wrap:wrap"><button class="btn-verify" onclick="openVerifyModal(\''+it.id+'\',\''+(it.isManual?'manual':'device')+'\',\''+esc(it.name).replace(/'/g,"\\'")+'\',\''+esc(it.tag).replace(/'/g,"\\'")+'\',\''+esc(it.location||it.branch).replace(/'/g,"\\'")+'\',\''+esc(it.pic||'-').replace(/'/g,"\\'")+'\',\''+it.condition+'\')">&#9989; Verifikasi</button>' +
        (it.isManual ? '<button class="btn-sm-action" onclick="openEditAssetModal(\''+it.id+'\')">&#9998;</button><button class="btn-sm-action" style="color:var(--red)" onclick="deleteManualAsset(\''+it.id+'\')">&#128465;</button>' : '') +
        '</div></td></tr>';
    }).join('') + '</tbody></table>';
  container.querySelectorAll('tbody tr').forEach(function(row, index) {
    var item = items[index];
    var asset = (item.isManual ? manualAssets : devices).find(function(a) { return a.id === item.id; });
    var actions = row.lastElementChild.firstElementChild;
    if (isAssetUser() || currentUser.role === 'viewer') actions.innerHTML = '';
    if (currentUser.role !== 'viewer') {
      if (isAssetUser() || !item.isManual) {
        var edit = document.createElement('button'); edit.className = 'btn-sm-action'; edit.textContent = 'Edit';
        edit.onclick = function() { if (item.isManual) openEditAssetModal(item.id); else openDeviceModal(item.id); }; actions.appendChild(edit);
      }
      var button = document.createElement('button'); button.className = 'btn-sm-action'; button.textContent = isAssetUser() ? 'Ajukan Switch' : 'Switch / Tugaskan';
      button.onclick = function() { requestAssetSwitch(item.isManual ? 'manual' : 'device', item.id); }; actions.appendChild(button);
    }
    var holder = document.createElement('div'); holder.textContent = 'Akun pemegang: ' + (asset.owner_username || 'Belum ditugaskan'); row.children[4].appendChild(holder);
    if (asset.recommendation) { var advice = document.createElement('p'); advice.className = 'asset-advice'; advice.textContent = asset.recommendation; row.children[1].appendChild(advice); }
  });
}

// ==================== MANUAL ASSET MODAL ====================

function openAddAssetModal() {
  document.getElementById('manualAssetModalTitle').textContent = 'Input Aset Manual Cabang';
  document.getElementById('assetEditId').value = '';
  document.getElementById('assetAcquisitionYear').value = '';
  document.getElementById('assetAcquisitionYear').disabled = false;
  document.getElementById('assetTagInput').value = '';
  document.getElementById('assetNameInput').value = '';
  document.getElementById('assetCategoryInput').value = 'pc';
  document.getElementById('assetBranchInput').value = (currentUser && currentUser.branch) ? currentUser.branch : ((document.getElementById('branchSelectFilter')||{}).value || '');
  if (currentUser && currentUser.role === 'adh') document.getElementById('assetBranchInput').disabled = true;
  else document.getElementById('assetBranchInput').disabled = false;
  document.getElementById('assetLocationInput').value = '';
  document.getElementById('assetAssignedInput').value = '';
  document.getElementById('assetSNInput').value = '';
  document.getElementById('assetConditionInput').value = 'good';
  document.getElementById('assetSpecsInput').value = '';
  document.getElementById('manualAssetModal').style.display = 'flex';
}

function openEditAssetModal(id) {
  var asset = manualAssets.find(function(a){return a.id===id});
  if (!asset) return;
  document.getElementById('manualAssetModalTitle').textContent = 'Edit Aset Manual';
  document.getElementById('assetEditId').value = asset.id;
  document.getElementById('assetAcquisitionYear').value = asset.acquisition_year || '';
  document.getElementById('assetAcquisitionYear').disabled = isAssetUser();
  ['assetTagInput','assetNameInput','assetCategoryInput','assetBranchInput','assetAssignedInput'].forEach(function(key) { document.getElementById(key).disabled = isAssetUser(); });
  document.getElementById('assetTagInput').value = asset.asset_tag;
  document.getElementById('assetNameInput').value = asset.name;
  document.getElementById('assetCategoryInput').value = asset.category || 'other';
  document.getElementById('assetBranchInput').value = asset.branch;
  if (currentUser && currentUser.role === 'adh') document.getElementById('assetBranchInput').disabled = true;
  else document.getElementById('assetBranchInput').disabled = false;
  document.getElementById('assetLocationInput').value = asset.location || '';
  document.getElementById('assetAssignedInput').value = asset.assigned_to || '';
  document.getElementById('assetSNInput').value = asset.serial_number || '';
  document.getElementById('assetConditionInput').value = asset.condition || 'good';
  document.getElementById('assetSpecsInput').value = asset.specs || '';
  document.getElementById('manualAssetModal').style.display = 'flex';
}

function closeManualAssetModal() {
  document.getElementById('manualAssetModal').style.display = 'none';
}

async function saveManualAsset() {
  var id = document.getElementById('assetEditId').value;
  var assetTag = document.getElementById('assetTagInput').value.trim();
  var name = document.getElementById('assetNameInput').value.trim();
  var category = document.getElementById('assetCategoryInput').value;
  var branch = document.getElementById('assetBranchInput').value.trim();
  if (currentUser && currentUser.role === 'adh' && currentUser.branch) branch = currentUser.branch;
  var location = document.getElementById('assetLocationInput').value.trim();
  var assignedTo = document.getElementById('assetAssignedInput').value.trim();
  var sn = document.getElementById('assetSNInput').value.trim();
  var condition = document.getElementById('assetConditionInput').value;
  var specs = document.getElementById('assetSpecsInput').value.trim();

  if (!assetTag || !name) { alert('Nomor tag dan Nama perangkat wajib diisi!'); return; }

  var payload = { asset_tag: assetTag, name: name, category: category, branch: branch || 'Pusat', location: location, assigned_to: assignedTo, serial_number: sn, condition: condition, specs: specs };
  payload.acquisition_year = Number(document.getElementById('assetAcquisitionYear').value) || 0;

  if (id) {
    var saved = await api('/api/assets/manual/' + id, { method: 'PUT', body: JSON.stringify(payload) });
    if (!saved || saved.error) { alert(saved && saved.error || 'Gagal menyimpan aset'); return; }
    showToast('Aset diperbarui');
  } else {
    var created = await api('/api/assets/manual', { method: 'POST', body: JSON.stringify(payload) });
    if (!created || created.error) { alert(created && created.error || 'Gagal menyimpan aset'); return; }
    showToast('Aset ditambahkan');
  }

  closeManualAssetModal();
  loadBranchAssets();
  loadBranches();
}

async function deleteManualAsset(id) {
  if (!confirm('Hapus data aset manual ini?')) return;
  var reason = prompt('Alasan penghapusan aset (wajib):');
  if (!reason || !reason.trim()) { alert('Alasan penghapusan wajib diisi.'); return; }
  var res = await api('/api/assets/manual/' + id, { method: 'DELETE', body: JSON.stringify({reason:reason.trim()}) });
  if (!res || res.error) { alert('Gagal menghapus: '+((res&&res.error)||'Terjadi kesalahan')); return; }
  showToast('Aset dihapus');
  loadBranchAssets();
}

// ==================== VERIFICATION MODAL ====================

function openVerifyModal(id, type, name, tag, location, pic, currentCondition) {
  document.getElementById('verifyAssetId').value = id;
  document.getElementById('verifyAssetTypeVal').value = type;
  document.getElementById('verifyAssetName').textContent = name;
  document.getElementById('verifyAssetTag').textContent = tag;
  document.getElementById('verifyAssetLoc').textContent = location + ' / ' + pic;
  document.getElementById('verifyAssetType').textContent = type === 'manual' ? 'Aset Manual' : 'Agent Monitor';
  document.getElementById('verifyStatusSelect').value = 'verified';
  document.getElementById('verifyConditionSelect').value = currentCondition || 'good';
  document.getElementById('verifyNotesInput').value = '';
  document.getElementById('verificationModal').style.display = 'flex';
}

function closeVerificationModal() {
  document.getElementById('verificationModal').style.display = 'none';
}

async function submitVerification() {
  var assetId = document.getElementById('verifyAssetId').value;
  var assetType = document.getElementById('verifyAssetTypeVal').value;
  var status = document.getElementById('verifyStatusSelect').value;
  var condition = document.getElementById('verifyConditionSelect').value;
  var notes = document.getElementById('verifyNotesInput').value.trim();

  var res = await api('/api/assets/verify', {
    method: 'POST',
    body: JSON.stringify({ asset_id: assetId, asset_type: assetType, status: status, condition: condition, notes: notes })
  });

  if (res && res.status === 'success') {
    showToast('Verifikasi fisik berhasil disimpan');
    closeVerificationModal();
    loadBranchAssets();
  } else {
    alert('Gagal: ' + ((res && res.error) || 'Terjadi kesalahan'));
  }
}

// ==================== CSV EXPORT ====================

function exportBranchAssetsCSV() {
  var branch = (document.getElementById('branchSelectFilter')||{}).value || '';
  if (currentUser && currentUser.role === 'adh') branch = currentUser.branch;

  var rows = [
    ['Tipe', 'Tag / Hostname', 'Nama Aset', 'Kategori', 'Cabang', 'Lokasi', 'PIC', 'Kondisi', 'Status Verifikasi', 'Diverifikasi Oleh', 'Tanggal', 'Catatan']
  ];
  manualAssets.forEach(function(a) {
    rows.push(['Manual', a.asset_tag, a.name, a.category, a.branch, a.location, a.assigned_to, a.condition, a.verification_status, a.verified_by, a.verified_at ? new Date(a.verified_at).toLocaleDateString() : '', a.verification_note]);
  });
  devices.forEach(function(d) {
    var dBranch = d.branch || d.group || 'default';
    if (branch && dBranch !== branch) return;
    rows.push(['Agent', d.hostname, d.os+' '+d.arch, 'pc', dBranch, d.local_ip||d.ip, d.note||'-', d.online?'good':'fair', d.verification_status||'unverified', d.verified_by, d.verified_at ? new Date(d.verified_at).toLocaleDateString() : '', d.verification_note]);
  });

  var csv = '\uFEFF';
  rows.forEach(function(r) { csv += r.map(function(v){ return '\"'+String(v||'').replace(/\"/g, '\"\"')+'\"'; }).join(',') + '\n'; });
  var blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  var url = URL.createObjectURL(blob);
  var a = document.createElement('a');
  a.href = url;
  a.download = 'verifikasi_aset_' + (branch||'semua') + '_' + new Date().toISOString().slice(0,10) + '.csv';
  a.click();
  URL.revokeObjectURL(url);
  showToast('Laporan CSV diexport');
}

// ==================== USER MANAGEMENT ====================

async function openUsersModal() { document.getElementById('usersModal').style.display = 'flex'; loadUsers(); }
function closeUsersModal() { document.getElementById('usersModal').style.display = 'none'; }

async function loadUsers() {
  var users = await api('/api/users');
  var container = document.getElementById('usersTableContainer');
  if (!container) return;
  if (!users || users.length === 0) { container.innerHTML = '<p style="color:var(--fg2);font-size:12px">Belum ada user terdaftar.</p>'; return; }
  container.innerHTML = '<table class="device-table"><thead><tr><th>Username</th><th>Role</th><th>Cabang</th><th>Aksi</th></tr></thead><tbody>' +
    users.map(function(u) {
      var isSelf = (currentUser && u.username === currentUser.username);
      var actions = [];
      if (isSelf) {
        actions.push('<button class="btn btn-ghost btn-sm" onclick="openChangeUsernameModal()">Ganti Nama</button>');
      } else {
        actions.push('<button class="btn btn-danger btn-sm" onclick="deleteUser('+u.id+', \''+esc(u.username)+'\')">Hapus</button>');
        actions.push('<button class="btn btn-ghost btn-sm" onclick="resetUserPassword('+u.id+', \''+esc(u.username)+'\')">Reset Pass</button>');
        if (u.mfa_enabled) {
          actions.push('<button class="btn btn-ghost btn-sm" onclick="resetUserMFA('+u.id+', \''+esc(u.username)+'\')">Reset 2FA</button>');
        }
      }
      return '<tr><td><strong>'+esc(u.username)+'</strong>'+(isSelf?' <small style="color:var(--accent)">(Anda)</small>':'')+(u.mfa_enabled?' <span class="badge-status verified" style="font-size:10px;padding:1px 6px">2FA ON</span>':'')+'</td><td><span class="user-badge '+u.role+'">'+esc(u.role)+'</span></td><td>'+esc(u.branch||'-')+'</td><td><div style="display:flex;gap:4px;flex-wrap:wrap">'+actions.join(' ')+'</div></td></tr>';
    }).join('') + '</tbody></table>';
}

async function createUser() {
  var username = document.getElementById('newUserUsername').value.trim();
  var password = document.getElementById('newUserPassword').value.trim();
  var role = document.getElementById('newUserRole').value;
  var branch = document.getElementById('newUserBranch').value.trim();
  if (!username || !password) { alert('Username dan Password wajib diisi'); return; }
  if ((role === 'adh' || role === 'user') && !branch) { alert('Nama lokasi wajib diisi untuk ADH/User'); return; }

  var res = await api('/api/users', { method:'POST', body:JSON.stringify({ username:username, password:password, role:role, branch:branch }) });
  if (res && res.status === 'created') {
    showToast('Akun berhasil dibuat');
    document.getElementById('newUserUsername').value = '';
    document.getElementById('newUserPassword').value = '';
    document.getElementById('newUserBranch').value = '';
    loadUsers();
    loadBranches();
  } else {
    alert('Gagal: ' + ((res && res.error) || 'Terjadi kesalahan'));
  }
}

async function deleteUser(id, username) {
  var nameStr = username ? 'user ' + username : 'akun ini';
  if (!confirm('Hapus ' + nameStr + '?')) return;
  var res = await api('/api/users/' + id, { method: 'DELETE' });
  if (res && res.status === 'deleted') {
    showToast('Akun berhasil dihapus');
    loadUsers();
  } else {
    alert('Gagal menghapus: ' + ((res && res.error) || 'Terjadi kesalahan'));
  }
}

// ==================== AUDIT HISTORY ====================

async function openHistoryModal() {
  document.getElementById('historyModal').style.display = 'flex';
  var branch = (document.getElementById('branchSelectFilter')||{}).value || '';
  if (currentUser && currentUser.role === 'adh') branch = currentUser.branch;
  var logs = await api('/api/assets/verifications?branch=' + encodeURIComponent(branch) + '&limit=40');
  var container = document.getElementById('historyTimeline');
  if (!container) return;
  if (!logs || logs.length === 0) { container.innerHTML = '<div class="empty-state"><p>Belum ada riwayat verifikasi fisik.</p></div>'; return; }
  container.innerHTML = logs.map(function(l) {
    var isV = l.status === 'verified';
    return '<div class="timeline-item '+(isV?'verified':'discrepancy')+'"><div class="timeline-meta"><strong>'+esc(l.verified_by)+'</strong> &bull; '+new Date(l.created_at).toLocaleString()+' &bull; Cabang: <strong>'+esc(l.branch)+'</strong></div><div>Status: <strong>'+(isV?'Terverifikasi':'Ada Masalah')+'</strong> | Perangkat: <strong>'+esc(l.asset_name||l.asset_id)+'</strong></div>'+(l.condition?'<div style="font-size:11px;color:var(--fg2)">Kondisi: '+esc(l.condition)+'</div>':'')+(l.notes?'<div style="margin-top:4px;padding:4px 8px;background:var(--bg);border-radius:4px">'+esc(l.notes)+'</div>':'')+'</div>';
  }).join('');
}

function closeHistoryModal() { document.getElementById('historyModal').style.display = 'none'; }

// ==================== DEVICE MODAL ====================

async function openDeviceModal(id) {
  var dev = await api('/api/devices/' + id);
  if (!dev) return;
  currentDevice = dev;
  document.getElementById('modalTitle').textContent = dev.hostname + ' - ' + dev.id;
  document.getElementById('modalDetails').innerHTML =
    '<div class="detail-item"><div class="label">OS</div><div class="value">'+esc(dev.os)+' '+esc(dev.arch)+'</div></div>' +
    '<div class="detail-item"><div class="label">IP Address</div><div class="value">'+esc(dev.ip)+' / '+esc(dev.local_ip)+'</div></div>' +
    '<div class="detail-item"><div class="label">CPU</div><div class="value">'+esc(dev.cpu_model)+' ('+dev.cpu_cores+' cores)</div></div>' +
    '<div class="detail-item"><div class="label">Memory</div><div class="value">'+fmtBytes(dev.memory_used)+' / '+fmtBytes(dev.memory_total)+'</div></div>' +
    '<div class="detail-item"><div class="label">Disk</div><div class="value">'+fmtBytes(dev.disk_used)+' / '+fmtBytes(dev.disk_total)+'</div></div>' +
    '<div class="detail-item"><div class="label">Status</div><div class="value"><span class="status-dot '+(dev.online?'online':'offline')+'"></span>'+(dev.online?'Online':'Offline')+'</div></div>' +
    '<div class="detail-item"><div class="label">Cabang</div><div class="value">'+esc(dev.branch||dev.group||'-')+'</div></div>' +
    '<div class="detail-item"><div class="label">Registered</div><div class="value">'+new Date(dev.registered_at).toLocaleString()+'</div></div>';
  document.getElementById('modalTags').value = dev.tags || '';
  document.getElementById('modalGroup').value = dev.group || 'default';
  document.getElementById('modalNote').value = dev.note || '';
  document.getElementById('modalAcquisitionYear').value = dev.acquisition_year || '';
  document.getElementById('modalAcquisitionYear').disabled = isAssetUser();
  document.getElementById('modalOwnership').textContent = 'Pemegang: ' + (dev.owner_username || 'Belum ditugaskan') + '. ' + (dev.recommendation || '') + ' ' + responsibilityNotice;
  document.getElementById('modalTags').disabled = isAssetUser();
  document.getElementById('modalGroup').disabled = isAssetUser();
  document.querySelectorAll('#deviceModal [onclick="deleteDevice()"], #deviceModal [onclick="startNetworkScan()"]').forEach(function(el) { el.style.display = isAssetUser() ? 'none' : ''; });
  document.getElementById('deviceModal').style.display = 'flex';
}

function closeDeviceModal() { document.getElementById('deviceModal').style.display = 'none'; currentDevice = null; }

async function startNetworkScan() {
  if (!currentDevice) return;
  if (!currentDevice.online) { alert('Agent harus online untuk memindai jaringan lokal.'); return; }
  if (!confirm('Pindai subnet lokal dari Agent ini? Scan dibatasi ke jaringan /24 Agent.')) return;
  var box = document.getElementById('networkScanResult');
  box.style.display = 'block'; box.innerHTML = '<small>Meminta Agent memindai jaringan lokal...</small>';
  var scan = await api('/api/network-scans', { method:'POST', body:JSON.stringify({device_id:currentDevice.id}) });
  if (!scan || scan.error) { box.innerHTML = '<small style="color:var(--danger)">Gagal: '+esc((scan&&scan.error)||'Terjadi kesalahan')+'</small>'; return; }
  var attempts = 0;
  var timer = setInterval(async function() {
    attempts++;
    var result = await api('/api/network-scans/' + encodeURIComponent(scan.id));
    if (!result) return;
    if (result.status === 'completed' || result.status === 'failed' || attempts >= 45) {
      clearInterval(timer);
      if (result.status !== 'completed') { box.innerHTML = '<small style="color:var(--danger)">Scan gagal atau timeout: '+esc(result.error||'Agent tidak merespons')+'</small>'; return; }
      var rows = (result.hosts||[]).map(function(h){ return '<tr><td>'+esc(h.ip)+'</td><td><code>'+esc(h.mac||'-')+'</code></td></tr>'; }).join('');
      box.innerHTML = '<div style="font-size:12px;margin-bottom:6px">Hasil scan '+esc(result.subnet)+' — '+(result.hosts||[]).length+' perangkat terdeteksi</div><table class="device-table"><thead><tr><th>IP Lokal</th><th>MAC Address</th></tr></thead><tbody>'+rows+'</tbody></table>';
    }
  }, 1000);
}

function resetIdleTimer() {
  if (!token) return;
  clearTimeout(idleTimer);
  idleTimer = setTimeout(function(){ alert('Sesi berakhir karena 30 menit tidak ada aktivitas.'); doLogout(); }, IDLE_TIMEOUT_MS);
}
['click','keydown','mousemove','touchstart'].forEach(function(e){ document.addEventListener(e, resetIdleTimer, {passive:true}); });

async function saveDeviceMeta() {
  if (!currentDevice) return;
  var saved = await api('/api/devices/' + currentDevice.id, {
    method: 'PUT',
    body: JSON.stringify({
      tags: document.getElementById('modalTags').value,
      group: document.getElementById('modalGroup').value,
      note: document.getElementById('modalNote').value,
      acquisition_year: Number(document.getElementById('modalAcquisitionYear').value) || 0
    })
  });
  if (!saved || saved.error) { alert(saved && saved.error || 'Gagal menyimpan perangkat'); return; }
  closeDeviceModal();
  loadDevices();
  loadGroups();
  loadBranchAssets();
  showToast('Device updated');
}

async function deleteDevice() {
  if (!currentDevice) return;
  if (!confirm('Delete device ' + currentDevice.hostname + '?')) return;
  var reason = prompt('Alasan penghapusan perangkat (wajib):');
  if (!reason || !reason.trim()) { alert('Alasan penghapusan wajib diisi.'); return; }
  var res = await api('/api/devices/' + currentDevice.id, { method: 'DELETE', body: JSON.stringify({reason:reason.trim()}) });
  if (!res || res.error) { alert('Gagal menghapus: '+((res&&res.error)||'Terjadi kesalahan')); return; }
  closeDeviceModal();
  loadDevices();
  loadBranchAssets();
  showToast('Device deleted');
}

// ==================== ASSET INVENTORY (GLOBAL) ====================

function renderAssets() {
  var search = ((document.getElementById('assetSearch')||{}).value || '').toLowerCase();
  var filtered = devices;
  if (search) {
    filtered = devices.filter(function(d){
      return (d.hostname||'').toLowerCase().includes(search) || (d.id||'').toLowerCase().includes(search) || (d.cpu_model||'').toLowerCase().includes(search) || (d.os||'').toLowerCase().includes(search) || (d.tags||'').toLowerCase().includes(search);
    });
  }
  if (filtered.length === 0) { document.getElementById('assetsTable').innerHTML = '<div class="empty-state"><p>No assets found</p></div>'; return; }
  document.getElementById('assetsTable').innerHTML = '<table class="device-table"><thead><tr><th>Hostname</th><th>ID</th><th>OS</th><th>CPU</th><th>RAM</th><th>Disk</th><th>IP</th><th>Group</th><th>Tags</th><th>Registered</th></tr></thead><tbody>' +
    filtered.map(function(d){
      return '<tr><td>'+esc(d.hostname)+'</td><td><code style="font-size:11px">'+esc(d.id)+'</code></td><td>'+esc(d.os)+'</td><td>'+esc(d.cpu_model||'-')+'</td><td>'+fmtBytes(d.memory_total)+'</td><td>'+fmtBytes(d.disk_total)+'</td><td>'+esc(d.ip)+'</td><td><span class="tag">'+esc(d.group||'default')+'</span></td><td>'+(d.tags||'')+'</td><td>'+new Date(d.registered_at).toLocaleDateString()+'</td></tr>';
    }).join('') + '</tbody></table>';
}

function exportAssets() {
  var rows = [['Hostname','ID','OS','Arch','CPU Model','RAM Total','Disk Total','IP','Group','Registered At']];
  devices.forEach(function(d){ rows.push([d.hostname, d.id, d.os, d.arch, d.cpu_model, fmtBytes(d.memory_total), fmtBytes(d.disk_total), d.ip, d.group, d.registered_at]); });
  var csv = '';
  rows.forEach(function(r){ csv += r.map(function(v){ return '\"'+String(v||'').replace(/\"/g, '\"\"')+'\"'; }).join(',') + '\n'; });
  var blob = new Blob([csv], { type: 'text/csv' });
  var url = URL.createObjectURL(blob);
  var a = document.createElement('a'); a.href = url; a.download = 'assets_' + new Date().toISOString().slice(0,10) + '.csv'; a.click(); URL.revokeObjectURL(url);
  showToast('CSV exported');
}

// ==================== REMOTE DESKTOP ====================

function updateRemoteDeviceList() {
  var sel = document.getElementById('remoteDeviceSelect');
  if (!sel) return;
  var cur = sel.value;
  sel.innerHTML = '<option value="">Select Device...</option>';
  devices.filter(function(d){return d.online}).forEach(function(d){ sel.innerHTML += '<option value="'+d.id+'">'+esc(d.hostname)+' ('+d.id+')</option>'; });
  if (cur) sel.value = cur;
}

function quickRemote(id) { showPage('remote'); document.getElementById('remoteDeviceSelect').value = id; startRemote(); }

function startRemote() {
  var deviceId = document.getElementById('remoteDeviceSelect').value;
  if (!deviceId) { showToast('Select a device first'); return; }
  setRemoteStatus('connecting');

  var sessionId = 'sess-' + Date.now().toString(36);
  var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  remoteWS = new WebSocket(proto + '//' + location.host + '/ws/relay/' + sessionId + '/viewer');

  var container = document.getElementById('remoteContainer');
  container.innerHTML = '<canvas id="remoteCanvas"></canvas>';
  var canvas = document.getElementById('remoteCanvas');
  remoteWS.binaryType = 'arraybuffer';

  remoteWS.onopen = function() {
    setRemoteStatus('connected');
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ action: 'signal', data: { type: 'start_relay', to: deviceId, payload: sessionId } }));
    }
  };

  remoteWS.onmessage = function(e) {
    if (e.data instanceof ArrayBuffer) {
      var blob = new Blob([e.data], { type: 'image/jpeg' });
      var url = URL.createObjectURL(blob);
      var img = new Image();
      img.onload = function() {
        canvas.width = img.width; canvas.height = img.height;
        var ctx = canvas.getContext('2d');
        ctx.drawImage(img, 0, 0);
        URL.revokeObjectURL(url);
      };
      img.src = url;
    }
  };

  remoteWS.onclose = function() { setRemoteStatus('disconnected'); };

  canvas.addEventListener('mousemove', function(e) {
    if (remoteWS && remoteWS.readyState === WebSocket.OPEN) {
      var rect = canvas.getBoundingClientRect();
      remoteWS.send(JSON.stringify({ type: 'mouse_move', x: (e.clientX - rect.left)/rect.width, y: (e.clientY - rect.top)/rect.height }));
    }
  });

  canvas.addEventListener('mousedown', function(e) {
    if (remoteWS && remoteWS.readyState === WebSocket.OPEN) {
      var rect = canvas.getBoundingClientRect();
      remoteWS.send(JSON.stringify({ type: 'mouse_click', x: (e.clientX - rect.left)/rect.width, y: (e.clientY - rect.top)/rect.height, button: e.button }));
    }
  });

  canvas.addEventListener('keydown', function(e) {
    if (remoteWS && remoteWS.readyState === WebSocket.OPEN) {
      remoteWS.send(JSON.stringify({ type: 'key_down', key: e.key, code: e.code }));
      e.preventDefault();
    }
  });

  canvas.tabIndex = 1; canvas.focus();
  document.getElementById('btnConnect').style.display = 'none';
  document.getElementById('btnDisconnect').style.display = '';
}

function stopRemote() {
  if (remoteWS) { remoteWS.close(); remoteWS = null; }
  setRemoteStatus('disconnected');
  document.getElementById('btnConnect').style.display = '';
  document.getElementById('btnDisconnect').style.display = 'none';
  document.getElementById('remoteContainer').innerHTML = '<div class="empty-state"><p>Disconnected</p></div>';
}

function setRemoteStatus(status) {
  var el = document.getElementById('remoteStatus');
  el.className = 'connection-status ' + status;
  el.textContent = status.charAt(0).toUpperCase() + status.slice(1);
}

function handleSignal(data) { console.log('Signal received:', data); }

// ==================== UTILS ====================

function esc(str) {
  if (!str) return '';
  var div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function fmtBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B';
  var units = ['B', 'KB', 'MB', 'GB', 'TB'];
  var i = Math.floor(Math.log(bytes) / Math.log(1024));
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i];
}

function timeAgo(dateStr) {
  if (!dateStr) return '-';
  var diff = (Date.now() - new Date(dateStr).getTime()) / 1000;
  if (diff < 60) return 'just now';
  if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
  if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
  return Math.floor(diff / 86400) + 'd ago';
}

function osIcon(os) {
  switch ((os || '').toLowerCase()) {
    case 'linux': return '&#128039;';
    case 'windows': return '&#127987;';
    case 'darwin': return '&#127822;';
    default: return '&#128187;';
  }
}

function showToast(msg) {
  var toast = document.createElement('div');
  toast.className = 'toast';
  toast.textContent = msg;
  document.body.appendChild(toast);
  setTimeout(function(){ toast.remove(); }, 3000);
}

// ==================== BOOT ====================
loadServerVersion();
checkAuth();

// ==================== RECONFIGURE ENDPOINT (ADMIN) ====================

async function openReconfigureModal() {
  var newUrl = prompt('Masukkan Endpoint Baru untuk 400 Agent (misal: https://newserver.synology.me:8443):');
  if (!newUrl || !newUrl.trim()) return;
  newUrl = newUrl.trim();
  if (!confirm('PERHATIAN: Semua ' + (devices.length || '?') + ' agent yang sedang online akan berpindah ke endpoint baru:\n\n' + newUrl + '\n\nApakah yakin?')) return;

  var res = await api('/api/agent/reconfigure', {
    method: 'POST',
    body: JSON.stringify({ server_url: newUrl })
  });

  if (res && res.status === 'reconfigure_sent') {
    showToast('Perintah pindah endpoint terkirim ke ' + res.agents_notified + ' agent. Config mereka akan otomatis terupdate.');
  } else {
    alert('Gagal: ' + ((res && res.error) || 'Unknown error'));
  }
}

// ==================== CHANGE & RESET PASSWORD ====================

function openChangePasswordModal() {
  var oldEl = document.getElementById('cpOldPass');
  var newEl = document.getElementById('cpNewPass');
  var confEl = document.getElementById('cpConfirmPass');
  var errEl = document.getElementById('cpError');
  if (oldEl) oldEl.value = '';
  if (newEl) newEl.value = '';
  if (confEl) confEl.value = '';
  if (errEl) { errEl.style.display = 'none'; errEl.textContent = ''; }
  var modal = document.getElementById('changePasswordModal');
  if (modal) modal.style.display = 'flex';
}

function closeChangePasswordModal() {
  var modal = document.getElementById('changePasswordModal');
  if (modal) modal.style.display = 'none';
}

async function submitChangePassword() {
  var oldPass = (document.getElementById('cpOldPass') || {}).value || '';
  var newPass = (document.getElementById('cpNewPass') || {}).value || '';
  var confirmPass = (document.getElementById('cpConfirmPass') || {}).value || '';
  var errEl = document.getElementById('cpError');

  function showErr(msg) {
    if (errEl) {
      errEl.textContent = msg;
      errEl.style.display = 'block';
    } else {
      alert(msg);
    }
  }

  if (!oldPass) { showErr('Password saat ini wajib diisi'); return; }
  if (!newPass || newPass.length < 8) { showErr('Password baru minimal 8 karakter'); return; }
  if (newPass !== confirmPass) { showErr('Konfirmasi password baru tidak cocok'); return; }
  if (newPass === oldPass) { showErr('Password baru tidak boleh sama dengan password lama'); return; }

  var res = await api('/api/auth/change-password', {
    method: 'POST',
    body: JSON.stringify({
      old_password: oldPass,
      new_password: newPass,
      confirm_password: confirmPass
    })
  });

  if (res && res.status === 'success') {
    closeChangePasswordModal();
    showToast('Password berhasil diubah!');
  } else {
    showErr((res && res.error) || 'Gagal mengubah password');
  }
}

async function resetUserPassword(id, username) {
  var newPass = prompt('Masukkan password baru untuk user ' + username + ' (minimal 8 karakter):');
  if (!newPass || !newPass.trim()) return;
  newPass = newPass.trim();
  if (newPass.length < 8) {
    alert('Password minimal 8 karakter!');
    return;
  }
  var res = await api('/api/users/' + id + '/reset-password', {
    method: 'POST',
    body: JSON.stringify({ new_password: newPass })
  });
  if (res && res.status === 'success') {
    showToast('Password untuk ' + username + ' berhasil direset');
  } else {
    alert('Gagal reset password: ' + ((res && res.error) || 'Terjadi kesalahan'));
  }
}

// ==================== 2FA / MFA FUNCTIONS ====================

async function doLoginMFA() {
  const ticket = (document.getElementById('loginMFATicket') || {}).value || '';
  const code = ((document.getElementById('loginMFACode') || {}).value || '').trim();
  const errEl = document.getElementById('loginError');

  if (!code || code.length !== 6) {
    if (errEl) {
      errEl.textContent = 'Masukkan 6 digit kode dari aplikasi Authenticator';
      errEl.style.display = 'block';
    }
    return;
  }

  try {
    const res = await fetch('/api/auth/login/mfa', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mfa_ticket: ticket, code: code, remember_device: !!(document.getElementById('rememberDevice') || {}).checked })
    });
    const data = await res.json();
    if (res.ok && data.token) {
      token = data.token;
      localStorage.setItem('rd_token', token);
      currentUser = { username: data.username, role: data.role, branch: data.branch };
      updateUserUI();
      document.getElementById('loginPage').style.display = 'none';
      document.getElementById('appContainer').style.display = 'flex';
      init();
    } else {
      if (errEl) {
        errEl.textContent = data.error || 'Kode 2FA salah / kedaluwarsa';
        errEl.style.display = 'block';
      }
    }
  } catch (e) {
    if (errEl) {
      errEl.textContent = 'Connection error';
      errEl.style.display = 'block';
    }
  }
}

function cancelMFA() {
  document.getElementById('loginMFA').style.display = 'none';
  document.getElementById('loginCreds').style.display = 'block';
  const errEl = document.getElementById('loginError');
  if (errEl) errEl.style.display = 'none';
}

let currentMFASetupSecret = '';

async function openMFAModal() {
  const modal = document.getElementById('mfaModal');
  if (!modal) return;
  modal.style.display = 'flex';

  const status = await api('/api/auth/mfa/status');
  if (status && status.mfa_enabled) {
    document.getElementById('mfaActiveSection').style.display = 'block';
    document.getElementById('mfaSetupSection').style.display = 'none';
    document.getElementById('mfaCloseAction').style.display = 'flex';
    document.getElementById('mfaDisablePass').value = '';
  } else {
    document.getElementById('mfaActiveSection').style.display = 'none';
    document.getElementById('mfaSetupSection').style.display = 'block';
    document.getElementById('mfaCloseAction').style.display = 'none';
    document.getElementById('mfaVerifyCode').value = '';
    const errEl = document.getElementById('mfaError');
    if (errEl) errEl.style.display = 'none';

    const setup = await api('/api/auth/mfa/setup', { method: 'POST' });
    if (setup && setup.secret) {
      currentMFASetupSecret = setup.secret;
      document.getElementById('mfaSecretText').textContent = setup.secret;
      const qrEl = document.getElementById('mfaQRCode');
      qrEl.innerHTML = '';
      if (typeof QRCode !== 'undefined') {
        new QRCode(qrEl, {
          text: setup.otpauth_url,
          width: 180,
          height: 180,
          colorDark: '#000000',
          colorLight: '#ffffff',
          correctLevel: QRCode.CorrectLevel.M
        });
      } else {
        qrEl.innerHTML = '<p style="color:#111;font-size:11px;padding:10px">Gunakan kunci manual di bawah untuk memasukkan ke Authenticator.</p>';
      }
    }
  }
}

function closeMFAModal() {
  const modal = document.getElementById('mfaModal');
  if (modal) modal.style.display = 'none';
}

function copyMFASecret() {
  const sec = document.getElementById('mfaSecretText').textContent;
  if (navigator.clipboard) {
    navigator.clipboard.writeText(sec).then(() => showToast('Kunci rahasia disalin ke clipboard'));
  } else {
    showToast('Kunci: ' + sec);
  }
}

async function submitEnableMFA() {
  const code = ((document.getElementById('mfaVerifyCode') || {}).value || '').trim();
  const errEl = document.getElementById('mfaError');
  if (!code || code.length !== 6) {
    if (errEl) {
      errEl.textContent = 'Masukkan 6 digit kode dari aplikasi Authenticator';
      errEl.style.display = 'block';
    }
    return;
  }

  const res = await api('/api/auth/mfa/enable', {
    method: 'POST',
    body: JSON.stringify({ secret: currentMFASetupSecret, code: code })
  });

  if (res && res.status === 'success') {
    closeMFAModal();
    showToast('2FA / MFA Berhasil Diaktifkan! Akun Anda sekarang aman.');
  } else {
    if (errEl) {
      errEl.textContent = (res && res.error) || 'Kode verifikasi salah, pastikan jam di HP Anda akurat.';
      errEl.style.display = 'block';
    }
  }
}

async function submitDisableMFA() {
  const pass = (document.getElementById('mfaDisablePass') || {}).value || '';
  if (!pass) {
    alert('Masukkan password akun Anda untuk menonaktifkan 2FA');
    return;
  }
  if (!confirm('Yakin ingin menonaktifkan 2FA? Keamanan akun akan berkurang.')) return;

  const res = await api('/api/auth/mfa/disable', {
    method: 'POST',
    body: JSON.stringify({ password: pass })
  });

  if (res && res.status === 'success') {
    closeMFAModal();
    showToast('2FA / MFA berhasil dinonaktifkan.');
  } else {
    alert('Gagal: ' + ((res && res.error) || 'Password salah'));
  }
}

async function resetUserMFA(id, username) {
  if (!confirm('Reset 2FA untuk user ' + username + '? User akan bisa login tanpa kode 2FA sampai dia mengaktifkannya lagi.')) return;
  const res = await api('/api/users/' + id + '/reset-mfa', { method: 'POST' });
  if (res && res.status === 'success') {
    showToast('2FA untuk ' + username + ' berhasil direset');
    loadUsers();
  } else {
    alert('Gagal reset 2FA: ' + ((res && res.error) || 'Terjadi kesalahan'));
  }
}

// ==================== CHANGE USERNAME ====================

function openChangeUsernameModal() {
  var unEl = document.getElementById('cuNewUsername');
  var pwEl = document.getElementById('cuPassword');
  var errEl = document.getElementById('cuError');
  if (unEl) unEl.value = (currentUser && currentUser.username) || '';
  if (pwEl) pwEl.value = '';
  if (errEl) { errEl.style.display = 'none'; errEl.textContent = ''; }
  var modal = document.getElementById('changeUsernameModal');
  if (modal) modal.style.display = 'flex';
}

function closeChangeUsernameModal() {
  var modal = document.getElementById('changeUsernameModal');
  if (modal) modal.style.display = 'none';
}

async function submitChangeUsername() {
  var newUn = ((document.getElementById('cuNewUsername') || {}).value || '').trim();
  var pw = ((document.getElementById('cuPassword') || {}).value || '');
  var errEl = document.getElementById('cuError');

  function showErr(msg) {
    if (errEl) {
      errEl.textContent = msg;
      errEl.style.display = 'block';
    } else {
      alert(msg);
    }
  }

  if (!newUn || newUn.length < 3) {
    showErr('Username minimal 3 karakter');
    return;
  }
  if (!pw) {
    showErr('Masukkan password saat ini sebagai konfirmasi');
    return;
  }

  var res = await api('/api/auth/change-username', {
    method: 'POST',
    body: JSON.stringify({ new_username: newUn, password: pw })
  });

  if (res && res.status === 'success') {
    closeChangeUsernameModal();
    if (res.token) {
      token = res.token;
      localStorage.setItem('rd_token', token);
    }
    if (currentUser) {
      currentUser.username = res.username;
      updateUserUI();
    }
    showToast('Username berhasil diubah menjadi: ' + res.username);
    if (document.getElementById('usersModal') && document.getElementById('usersModal').style.display !== 'none') {
      loadUsers();
    }
  } else {
    showErr((res && res.error) || 'Gagal mengubah username');
  }
}

// ==================== SECURITY AUDIT LOGS ====================

async function openSecurityLogsModal() {
  var modal = document.getElementById('securityLogsModal');
  if (modal) modal.style.display = 'flex';
  loadSecurityLogs();
}

function closeSecurityLogsModal() {
  var modal = document.getElementById('securityLogsModal');
  if (modal) modal.style.display = 'none';
}

async function loadSecurityLogs() {
  var container = document.getElementById('securityLogsContainer');
  if (!container) return;
  container.innerHTML = '<p style="color:var(--fg2);font-size:12px;padding:12px">Memuat log audit...</p>';

  var logs = await api('/api/auth/logs?limit=50');
  if (!logs || logs.length === 0) {
    container.innerHTML = '<div class="empty-state"><p>Belum ada catatan aktivitas login.</p></div>';
    return;
  }

  container.innerHTML = '<table class="device-table"><thead><tr><th>Waktu</th><th>Username</th><th>Alamat IP</th><th>Status</th><th>Keterangan</th></tr></thead><tbody>' +
    logs.map(function(l) {
      var badgeClass = 'unverified';
      var label = 'GAGAL';
      if (l.status === 'success') {
        badgeClass = 'verified';
        label = '&#10004; BERHASIL';
      } else if (l.status === 'blocked') {
        badgeClass = 'discrepancy';
        label = '&#9888; BLOCKED (15m)';
      } else if (l.status === 'mfa_failed') {
        badgeClass = 'discrepancy';
        label = '&#10060; 2FA SALAH';
      } else {
        badgeClass = 'discrepancy';
        label = '&#10060; GAGAL';
      }

      return '<tr>' +
        '<td><small>' + new Date(l.created_at).toLocaleString() + '</small></td>' +
        '<td><strong>' + esc(l.username) + '</strong></td>' +
        '<td><code>' + esc(l.ip) + '</code></td>' +
        '<td><span class="badge-status ' + badgeClass + '">' + label + '</span></td>' +
        '<td><small style="color:var(--fg2)">' + esc(l.reason || '-') + '</small></td>' +
      '</tr>';
    }).join('') + '</tbody></table>';
}

function switchSecTab(tab) {
  var logsBtn = document.getElementById('tabBtnSecLogs');
  var setBtn = document.getElementById('tabBtnSecSettings');
  var logsPanel = document.getElementById('secLogsPanel');
  var setPanel = document.getElementById('secSettingsPanel');

  if (tab === 'logs') {
    if (logsBtn) logsBtn.classList.add('active');
    if (setBtn) setBtn.classList.remove('active');
    if (logsPanel) logsPanel.style.display = 'block';
    if (setPanel) setPanel.style.display = 'none';
    loadSecurityLogs();
  } else {
    if (logsBtn) logsBtn.classList.remove('active');
    if (setBtn) setBtn.classList.add('active');
    if (logsPanel) logsPanel.style.display = 'none';
    if (setPanel) setPanel.style.display = 'block';
    loadSecuritySettings();
  }
}

async function loadSecuritySettings() {
  var res = await api('/api/security/settings');
  if (!res) return;

  var s = res.settings || {};
  var enCb = document.getElementById('secRateLimitEnabled');
  var maxInput = document.getElementById('secMaxAttempts');
  var blockInput = document.getElementById('secBlockDuration');
  var wlInput = document.getElementById('secIPWhitelist');

  if (enCb) enCb.checked = !!s.rate_limit_enabled;
  if (maxInput) maxInput.value = s.max_login_attempts || 5;
  if (blockInput) blockInput.value = s.block_duration_minutes || 15;
  if (wlInput) wlInput.value = s.ip_whitelist || '';

  // Render blocked IPs
  var bContainer = document.getElementById('blockedIPsContainer');
  if (!bContainer) return;
  var blocked = res.blocked_ips || [];
  if (blocked.length === 0) {
    bContainer.innerHTML = '<div style="background:var(--bg);padding:12px;border-radius:var(--radius);text-align:center;color:var(--fg2);font-size:12px">Tidak ada IP yang sedang diblokir saat ini.</div>';
    return;
  }

  bContainer.innerHTML = '<table class="device-table"><thead><tr><th>Alamat IP</th><th>Salah Password</th><th>Waktu Diblokir</th><th>Sisa Blokir</th><th>Aksi</th></tr></thead><tbody>' +
    blocked.map(function(b) {
      return '<tr>' +
        '<td><code>' + esc(b.ip) + '</code></td>' +
        '<td>' + b.failed_count + ' kali</td>' +
        '<td><small>' + new Date(b.blocked_at).toLocaleTimeString() + '</small></td>' +
        '<td><span class="badge-status discrepancy">~' + b.minutes_left + ' menit lagi</span></td>' +
        '<td><button class="btn btn-primary btn-sm" onclick="unblockIP(\'' + esc(b.ip) + '\')">Buka Blokir</button></td>' +
      '</tr>';
    }).join('') + '</tbody></table>';
}

async function saveSecuritySettings() {
  var en = (document.getElementById('secRateLimitEnabled') || {}).checked;
  var maxAtt = parseInt((document.getElementById('secMaxAttempts') || {}).value) || 5;
  var blockDur = parseInt((document.getElementById('secBlockDuration') || {}).value) || 15;
  var wl = ((document.getElementById('secIPWhitelist') || {}).value || '').trim();

  if (maxAtt <= 0 || maxAtt > 50) {
    alert('Batas maksimal salah password harus antara 1 sampai 50');
    return;
  }
  if (blockDur <= 0 || blockDur > 1440) {
    alert('Durasi pemblokiran harus antara 1 sampai 1440 menit (24 jam)');
    return;
  }

  var res = await api('/api/security/settings', {
    method: 'POST',
    body: JSON.stringify({
      rate_limit_enabled: en,
      max_login_attempts: maxAtt,
      block_duration_minutes: blockDur,
      ip_whitelist: wl
    })
  });

  if (res && res.status === 'success') {
    showToast('Pengaturan proteksi brute-force berhasil disimpan!');
    loadSecuritySettings();
  } else {
    alert('Gagal menyimpan: ' + ((res && res.error) || 'Terjadi kesalahan'));
  }
}

async function unblockIP(ip) {
  if (!confirm('Buka blokir IP ' + ip + '? User di IP ini akan bisa mencoba login kembali segera.')) return;
  var res = await api('/api/security/unblock', {
    method: 'POST',
    body: JSON.stringify({ ip: ip })
  });

  if (res && res.status === 'success') {
    showToast('IP ' + ip + ' berhasil dibuka blokirnya!');
    loadSecuritySettings();
  } else {
    alert('Gagal membuka blokir: ' + ((res && res.error) || 'Terjadi kesalahan'));
  }
}

// ==================== LOCATION MANAGEMENT ====================

var branchTypeLabels = {};
var branchManagementRows = {};

async function openBranchesModal() {
  var modal = document.getElementById('branchesModal');
  if (!modal) return;
  modal.style.display = 'flex';
  await loadBusinessUnits();
  await loadBranchesManagement();
}

async function loadBusinessUnits(selected) {
  var box = document.getElementById('branchBusinessUnitsInput'); if (!box) return;
  var units = await api('/api/business-units') || []; selected = selected || [];
  box.innerHTML = units.map(function(u){ return '<label style="display:flex;gap:4px;align-items:center;font-size:12px"><input type="checkbox" value="'+esc(u)+'" '+(selected.indexOf(u)>=0?'checked':'')+'>'+esc(u)+'</label>'; }).join('') || '<small style="color:var(--fg2)">Belum ada bisnis unit.</small>';
}
function selectedBusinessUnits() { return Array.prototype.slice.call(document.querySelectorAll('#branchBusinessUnitsInput input:checked')).map(function(x){return x.value;}); }
async function addBusinessUnit() { var name=prompt('Nama bisnis unit baru:'); if(!name) return; var r=await api('/api/business-units',{method:'POST',body:JSON.stringify({name:name.trim()})}); if(!r||r.error){alert('Gagal: '+((r&&r.error)||'Terjadi kesalahan'));return;} await loadBusinessUnits(selectedBusinessUnits().concat([name.trim()])); }

function closeBranchesModal() {
  var modal = document.getElementById('branchesModal');
  if (modal) modal.style.display = 'none';
}

async function renameLocationType() {
  var oldType = prompt('Nama tipe saat ini, contoh: Pusat');
  if (oldType === null) return;
  oldType = oldType.trim();
  var newType = prompt('Nama tipe baru, contoh: HO', oldType);
  if (newType === null) return;
  newType = newType.trim();
  if (!oldType || !newType) { alert('Nama tipe wajib diisi.'); return; }
  if (!confirm('Ubah semua tipe lokasi "' + oldType + '" menjadi "' + newType + '"?')) return;
  var res = await api('/api/location-types/rename', {method:'POST', body:JSON.stringify({old_type:oldType, new_type:newType})});
  if (!res || res.error) { alert('Gagal mengubah tipe: ' + ((res&&res.error)||'Terjadi kesalahan')); return; }
  showToast('Nama tipe lokasi berhasil diperbarui.');
  await loadBranchesManagement();
}

async function loadBranchesManagement() {
  var container = document.getElementById('branchesTableContainer');
  if (!container) return;
  container.innerHTML = '<div style="padding:12px;color:var(--fg2);font-size:12px">Memuat lokasi...</div>';
  var branches = await api('/api/branches?detail=true');
  if (!Array.isArray(branches)) {
    container.innerHTML = '<div style="padding:12px;color:var(--danger);font-size:12px">Gagal memuat daftar lokasi.</div>';
    return;
  }
  if (!branches.length) {
    container.innerHTML = '<div style="padding:12px;color:var(--fg2);font-size:12px">Belum ada lokasi tersimpan.</div>';
    return;
  }
  branchManagementRows = {};
  branches.forEach(function(b) { branchManagementRows[b.id] = b; });
  container.innerHTML = '<table class="device-table"><thead><tr><th>Nama Lokasi</th><th>Tipe</th><th>Bisnis Unit</th><th>Aksi</th></tr></thead><tbody>' + branches.map(function(b) {
    var isPusat = b.name === 'Pusat';
    return '<tr><td><strong>' + esc(b.name) + '</strong></td><td><span class="tag">' + esc(branchTypeLabels[b.type] || b.type) + '</span></td><td>' + esc((b.business_units || []).join(', ') || '-') + '</td><td><button class="btn btn-ghost btn-sm" onclick="editBranch(' + b.id + ')">Ubah</button> ' + (isPusat ? '' : '<button class="btn btn-danger btn-sm" onclick="deleteBranch(' + b.id + ')">Hapus</button>') + '</td></tr>';
  }).join('') + '</tbody></table>';
}

async function createBranch() {
  var nameInput = document.getElementById('branchNameInput');
  var typeInput = document.getElementById('branchTypeInput');
  var name = (nameInput.value || '').trim();
  if (!name) { alert('Nama lokasi wajib diisi.'); nameInput.focus(); return; }
  var businessUnits = selectedBusinessUnits();
  var res = await api('/api/branches', { method:'POST', body:JSON.stringify({ name:name, type:typeInput.value, business_units:businessUnits }) });
  if (!res || res.error) { alert('Gagal menambahkan lokasi: ' + ((res && res.error) || 'Terjadi kesalahan')); return; }
  nameInput.value = '';
  showToast('Lokasi ' + name + ' berhasil ditambahkan.');
  await Promise.all([loadBranchesManagement(), loadBranches()]);
}

async function editBranch(id) {
  var branch = branchManagementRows[id];
  if (!branch) return;
  var oldName = branch.name;
  var oldType = branch.type;
  await loadBusinessUnits(branch.business_units || []);
  var name = prompt('Nama lokasi:', oldName);
  if (name === null) return;
  name = name.trim();
  if (!name) { alert('Nama lokasi wajib diisi.'); return; }
  var type = prompt('Tipe lokasi:', oldType);
  if (type === null) return;
  var res = await api('/api/branches/' + id, { method:'PUT', body:JSON.stringify({ name:name, type:type.trim(), business_units:selectedBusinessUnits() }) });
  if (!res || res.error) { alert('Gagal mengubah lokasi: ' + ((res && res.error) || 'Terjadi kesalahan')); return; }
  showToast('Lokasi berhasil diperbarui.');
  await Promise.all([loadBranchesManagement(), loadBranches(), loadBranchAssets()]);
}

async function deleteBranch(id) {
  var branch = branchManagementRows[id];
  if (!branch) return;
  var name = branch.name;
  if (!confirm('Hapus lokasi ' + name + '? Lokasi yang masih dipakai tidak dapat dihapus.')) return;
  var res = await api('/api/branches/' + id, { method:'DELETE' });
  if (!res || res.error) { alert('Gagal menghapus lokasi: ' + ((res && res.error) || 'Terjadi kesalahan')); return; }
  showToast('Lokasi ' + name + ' berhasil dihapus.');
  await Promise.all([loadBranchesManagement(), loadBranches()]);
}

// ==================== DOWNLOAD AGENT PACKAGE ====================

async function openDownloadAgentModal() {
  var modal = document.getElementById('downloadAgentModal');
  if (!modal) return;

  var sel = document.getElementById('dlAgentBranch');
  if (sel) {
    var branches = await api('/api/branches') || [];
    if (currentUser && currentUser.role === 'adh' && currentUser.branch) {
      sel.innerHTML = '<option value="' + esc(currentUser.branch) + '">' + esc(currentUser.branch) + ' (Cabang Anda)</option>';
      sel.value = currentUser.branch;
      sel.disabled = true;
    } else {
      sel.disabled = false;
      var opts = ['Pusat'];
      branches.forEach(function(b) {
        if (b && opts.indexOf(b) === -1) opts.push(b);
      });
      sel.innerHTML = '';
      opts.forEach(function(b) {
        sel.innerHTML += '<option value="' + esc(b) + '">' + esc(b) + '</option>';
      });
      var currentFilter = (document.getElementById('branchSelectFilter') || {}).value;
      if (currentFilter && opts.indexOf(currentFilter) !== -1) {
        sel.value = currentFilter;
      } else {
        sel.value = 'Pusat';
      }
    }
  }

  modal.style.display = 'flex';
}

function closeDownloadAgentModal() {
  var modal = document.getElementById('downloadAgentModal');
  if (modal) modal.style.display = 'none';
}

function submitDownloadAgentPackage() {
  var sel = document.getElementById('dlAgentBranch');
  var branch = (sel && sel.value) ? sel.value.trim() : 'Pusat';
  if (currentUser && currentUser.role === 'adh' && currentUser.branch) {
    branch = currentUser.branch;
  }
  var os = (document.getElementById('dlAgentOS') || {}).value || 'windows';

  var downloadURL = '/api/agent/package?branch=' + encodeURIComponent(branch) + '&os=' + encodeURIComponent(os) + '&token=' + encodeURIComponent(token);
  
  showToast('Menyiapkan paket installer untuk cabang ' + branch + '...');
  
  var a = document.createElement('a');
  a.href = downloadURL;
  a.setAttribute('download', 'RemoteDesk-Agent-' + branch + '.zip');
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);

  setTimeout(function() {
    closeDownloadAgentModal();
  }, 1000);
}

