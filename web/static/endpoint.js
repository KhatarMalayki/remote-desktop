let endpointSelectedDevice = null;
let endpointReturnFocus = null;

function endpointBadges(device) {
  const state = device.endpoint || {};
  const stale = !device.online || !state.checked_at || Date.now() / 1000 - state.checked_at > 900;
  const badges = (state.applications || []).map(app => {
    let icon = '⚪';
    let iconColor = '#94a3b8';
    let badgeBg = 'rgba(160,163,177,0.12)';
    let badgeBorder = '1px solid rgba(160,163,177,0.25)';
    let statusText = 'status belum diketahui';

    if (app.status === 'detected') {
      icon = '✔';
      iconColor = '#10b981';
      badgeBg = 'rgba(16,185,129,0.12)';
      badgeBorder = '1px solid rgba(16,185,129,0.3)';
      statusText = 'terpasang';
    } else if (app.status === 'not_detected') {
      icon = '✖';
      iconColor = '#ef4444';
      badgeBg = 'rgba(239,68,68,0.1)';
      badgeBorder = '1px solid rgba(239,68,68,0.2)';
      statusText = 'tidak terpasang';
    }

    const title = esc(app.label + ': ' + statusText + (stale ? ' (data lama/offline)' : ''));
    return '<span style="display:inline-flex;align-items:center;gap:3px;padding:2px 6px;border-radius:4px;font-size:11px;font-weight:600;background:' + badgeBg + ';border:' + badgeBorder + ';white-space:nowrap" title="' + title + '">' +
      '<span style="font-weight:bold;color:' + iconColor + '">' + icon + '</span> ' +
      esc(app.label) +
    '</span>';
  }).join('');

  return '<div style="display:flex;flex-wrap:wrap;gap:4px;align-items:center;margin-bottom:6px">' + (badges || '<span style="color:var(--fg2);font-size:11px">-</span>') + '</div>' +
    '<button class="btn btn-ghost btn-sm" style="width:100%;white-space:nowrap;justify-content:center;padding:2px 6px;font-size:11px" data-device="' + esc(device.id) + '" onclick="openEndpointDevice(this.dataset.device)">🛡️ Kelola Policy</button>';
}

function updateEndpointDevicePicker(activeId) {
  const select = document.getElementById('endpointDevicePicker');
  if (!select) return;
  select.replaceChildren();
  const list = (typeof devices !== 'undefined' && Array.isArray(devices)) ? devices : [];
  list.forEach(d => {
    const opt = document.createElement('option');
    opt.value = d.id;
    opt.textContent = (d.hostname || d.id) + (d.online ? ' (Online)' : ' (Offline)') + (d.assigned_to ? ' — ' + d.assigned_to : '');
    if (d.id === activeId) opt.selected = true;
    select.append(opt);
  });
}

function onEndpointDevicePickerChange(id) {
  if (!id || id === endpointSelectedDevice) return;
  openEndpointDevice(id);
}

async function openPolicyManagerModal() {
  if (!currentUser || currentUser.role !== 'admin') return appAlert('Hanya admin dapat mengelola policy agent.');
  const list = (typeof devices !== 'undefined' && Array.isArray(devices)) ? devices : [];
  if (list.length === 0) return appAlert('Belum ada perangkat terdaftar.');
  const targetId = endpointSelectedDevice || (list.find(d => d.online) || list[0]).id;
  await openEndpointDevice(targetId);
}

function showEndpointModal(id) {
  endpointReturnFocus = document.activeElement;
  const modal = document.getElementById(id);
  modal.style.display = 'flex';
  const control = modal.querySelector('input,select,button');
  if (control) control.focus();
}

function closeEndpointModal(id) {
  document.getElementById(id).style.display = 'none';
  if (endpointReturnFocus && endpointReturnFocus.isConnected) endpointReturnFocus.focus();
}

document.addEventListener('keydown', function(event) {
  const modal = ['endpointAppsModal', 'endpointDeviceModal'].map(id => document.getElementById(id)).find(item => item && item.style.display === 'flex');
  if (!modal) return;
  if (event.key === 'Escape') { event.preventDefault(); closeEndpointModal(modal.id); }
  if (event.key !== 'Tab') return;
  const controls = Array.from(modal.querySelectorAll('button,input,select')).filter(item => !item.disabled && item.getClientRects().length);
  const first = controls[0], last = controls[controls.length - 1];
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
});

function addTrackedApplicationRow(app) {
  app = app || {label:'', match:''};
  const row = document.createElement('div'); row.className = 'form-group';
  const label = document.createElement('input'); label.value = app.label; label.maxLength = 24; label.setAttribute('aria-label', 'Label badge'); label.placeholder = 'Label, contoh: ME'; label.dataset.field = 'label';
  const match = document.createElement('input'); match.value = app.match; match.maxLength = 256; match.setAttribute('aria-label', 'Nama aplikasi atau service yang dicocokkan'); match.placeholder = 'Contoh: ManageEngine|Endpoint Central'; match.dataset.field = 'match';
  const remove = document.createElement('button'); remove.className = 'btn btn-ghost'; remove.textContent = 'Hapus dari pemantauan'; remove.onclick = () => row.remove();
  row.append(label, match, remove); document.getElementById('endpointAppsRows').append(row);
}

async function openTrackedApplications() {
  if (!currentUser || currentUser.role !== 'admin') return appAlert('Hanya admin dapat mengubah daftar global.');
  const apps = await api('/api/endpoint/applications');
  if (!Array.isArray(apps)) return appAlert(apps && apps.error || 'Daftar aplikasi gagal dimuat');
  document.getElementById('endpointAppsRows').replaceChildren();
  apps.forEach(addTrackedApplicationRow);
  document.getElementById('endpointAppsMessage').textContent = '';
  showEndpointModal('endpointAppsModal');
}

async function saveTrackedApplications() {
  const apps = Array.from(document.querySelectorAll('#endpointAppsRows > div'), row => ({label:row.querySelector('[data-field="label"]').value.trim(), match:row.querySelector('[data-field="match"]').value.trim()}));
  if (!await appConfirm('Simpan daftar pemantauan untuk SEMUA perangkat? Tidak memasang atau menghapus software.')) return;
  const button = document.getElementById('endpointAppsSave'); button.disabled = true;
  try {
    const result = await api('/api/endpoint/applications', {method:'PUT', body:JSON.stringify(apps)});
    document.getElementById('endpointAppsMessage').textContent = result && result.status === 'ok' ? 'Daftar tersimpan. Badge dihitung ulang dari inventaris terakhir.' : result && result.error || 'Gagal menyimpan';
    if (result && result.status === 'ok') await loadDevices();
  } finally { button.disabled = false; }
}

async function openEndpointDevice(id) {
  endpointSelectedDevice = id;
  updateEndpointDevicePicker(id);
  document.getElementById('endpointPolicyMode').value = 'audit';
  document.getElementById('endpointPolicyMessage').textContent = '';
  await refreshEndpointDevice();
  showEndpointModal('endpointDeviceModal');
}

async function refreshEndpointDevice() {
  const id = endpointSelectedDevice;
  const device = await api('/api/devices/' + encodeURIComponent(id));
  if (id !== endpointSelectedDevice) return;
  const output = document.getElementById('endpointDeviceInfo');
  output.replaceChildren();

  if (!device || device.error) {
    const err = document.createElement('div');
    err.className = 'endpoint-callout danger';
    err.textContent = '⚠️ ' + (device && device.error || 'Gagal membaca data perangkat.');
    output.append(err);
    document.getElementById('endpointPolicyControls').style.display = 'none';
    return;
  }

  const state = device.endpoint || {}, report = state.report || {}, policy = state.policy;
  document.getElementById('endpointDeviceTitle').textContent = '🛡️ ' + (device.hostname || device.id) + ' — Kelola Policy Agent';

  // 1. Overview Bar
  const overview = document.createElement('div');
  overview.className = 'endpoint-overview-bar';
  const onlineTag = device.online
    ? '<span style="color:#10b981;font-weight:600">● Online</span>'
    : '<span style="color:var(--fg2);font-weight:600">○ Offline</span>';
  const checkedTime = state.checked_at
    ? new Date(state.checked_at * 1000).toLocaleString()
    : 'Belum ada scan (perlu agent 0.2.58+)';
  const assigned = device.assigned_to ? ' • PIC: <strong>' + esc(device.assigned_to) + '</strong>' : '';
  const agentVer = device.version ? ' • Agent: v' + esc(device.version) : '';
  overview.innerHTML = '<div>' + onlineTag + ' • <strong>' + esc(device.hostname || device.id) + '</strong>' + assigned + agentVer + '</div>' +
    '<div style="color:var(--fg2);font-size:11px">Pemeriksaan: ' + esc(checkedTime) + '</div>';
  output.append(overview);

  // 2. Apps Header
  const appsHeader = document.createElement('div');
  appsHeader.style.cssText = 'display:flex;justify-content:space-between;align-items:center;margin:14px 0 6px';
  appsHeader.innerHTML = '<span style="font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:0.5px;color:var(--fg)">Status Aplikasi Terpantau</span>' +
    '<button class="btn btn-ghost btn-sm" onclick="openTrackedApplications()" style="padding:2px 8px;font-size:11px">⚙️ Atur Target Aplikasi</button>';
  output.append(appsHeader);

  // 3. Grid Apps
  const grid = document.createElement('div');
  grid.className = 'endpoint-apps-grid';
  const apps = state.applications || [];
  if (apps.length === 0) {
    const empty = document.createElement('div');
    empty.style.cssText = 'color:var(--fg2);font-size:12px;padding:8px 0';
    empty.textContent = 'Belum ada aplikasi yang dipantau.';
    grid.append(empty);
  } else {
    apps.forEach(app => {
      const card = document.createElement('div');
      const isDetected = app.status === 'detected';
      card.className = 'endpoint-app-card' + (isDetected ? ' detected' : '');

      let badgeHtml = '';
      if (isDetected) {
        badgeHtml = '<span class="badge badge-green"><span style="color:#10b981;font-weight:bold;margin-right:3px">✔</span>Terpasang</span>';
      } else if (app.status === 'not_detected') {
        badgeHtml = '<span class="badge badge-gray"><span style="color:#ef4444;font-weight:bold;margin-right:3px">✖</span>Tidak Ada</span>';
      } else {
        badgeHtml = '<span class="badge badge-gray"><span style="color:#94a3b8;font-weight:bold;margin-right:3px">⚪</span>Belum Scan</span>';
      }

      let matchesHtml = '';
      if (Array.isArray(app.matches) && app.matches.length > 0) {
        matchesHtml = app.matches.map(m => esc(m.name + (m.version ? ' v' + m.version : '') + (m.service ? ' (service: ' + m.service + ')' : ''))).join('<br>');
      } else if (isDetected) {
        matchesHtml = 'Terdeteksi pada sistem';
      } else {
        matchesHtml = 'Tidak ditemukan di registry machine atau service';
      }

      card.innerHTML = '<div class="app-header">' +
        '<span class="app-name">' + esc(app.label) + '</span>' +
        badgeHtml +
      '</div>' +
      '<div class="app-meta">' + matchesHtml + '</div>';
      grid.append(card);
    });
  }
  output.append(grid);

  // 4. Panel Detail Ringkasan Status
  const summaryPanel = document.createElement('div');
  summaryPanel.className = 'detail-grid';
  summaryPanel.style.cssText = 'background:var(--bg2);border:1px solid var(--border);border-radius:var(--radius);padding:14px;margin-bottom:12px';

  let lockVal = 'Belum diketahui';
  if (state.report && report.status !== 'unsupported') {
    lockVal = report.lock_present ? (Math.round(report.lock_seconds / 60) + ' Menit (' + report.lock_seconds + ' detik)') : 'Belum diatur';
  }

  let jobVal = 'Belum ada job';
  if (policy) {
    const jobColor = policy.status === 'applied' ? '#10b981' : policy.status === 'conflict' ? '#ef4444' : '#fbbf24';
    jobVal = '<span style="color:' + jobColor + ';font-weight:600">' + esc(policy.status.toUpperCase()) + '</span> — ' + esc(policy.detail || '');
  }

  summaryPanel.innerHTML = '<div class="detail-item">' +
    '<div class="label">Penguncian Layar Saat Ini</div>' +
    '<div class="value" style="font-weight:600;font-size:13px">' + esc(lockVal) + '</div>' +
  '</div>' +
  '<div class="detail-item">' +
    '<div class="label">Job Policy Terakhir</div>' +
    '<div class="value" style="font-size:12px">' + jobVal + '</div>' +
  '</div>';
  output.append(summaryPanel);

  // 5. Warning Callouts (Domain Restriction & Drift)
  if (report.policy_error) {
    const warn = document.createElement('div');
    warn.className = 'endpoint-callout warning';
    warn.innerHTML = '<span style="font-size:16px">⚠️</span><div><strong>Pembatasan Kebijakan:</strong> ' + esc(report.policy_error) + '</div>';
    output.append(warn);
  }

  if (report.drift) {
    const driftWarn = document.createElement('div');
    driftWarn.className = 'endpoint-callout danger';
    driftWarn.innerHTML = '<span style="font-size:16px">⚠️</span><div><strong>Peringatan Drift:</strong> Pengaturan registry penguncian layar telah berubah di luar RemoteDesk. Sistem tidak akan menimpa otomatis.</div>';
    output.append(driftWarn);
  }

  // 6. Policy Controls Visibility
  const supported = device.os === 'windows' && !isVersionNewer('0.2.58', device.version || '');
  const controls = document.getElementById('endpointPolicyControls');
  controls.style.display = currentUser && currentUser.role === 'admin' ? 'block' : 'none';
  const sendBtn = document.getElementById('endpointPolicySend');
  sendBtn.disabled = !supported || !!(policy && ['pending','delivered'].includes(policy.status));

  if (!supported) {
    const notSup = document.createElement('div');
    notSup.className = 'endpoint-callout info';
    notSup.innerHTML = 'ℹ️ Fitur penerapan policy membutuhkan perangkat Windows dengan agent minimal v0.2.58.';
    output.append(notSup);
  }
}

async function submitLockPolicy() {
  const mode = document.getElementById('endpointPolicyMode').value;
  const minutes = Number(document.getElementById('endpointPolicyMinutes').value);
  if (!Number.isInteger(minutes) || minutes < 1 || minutes > 1440) return appAlert('Waktu harus 1–1440 menit.');
  const device = endpointSelectedDevice;
  if (!await appConfirm('Operasi ' + mode + ' untuk ' + device + ', target ' + minutes + ' menit.\n' + (mode === 'audit' ? 'Tidak mengubah pengaturan.' : 'Mengubah policy lock-screen. Backup dan pemeriksaan konflik dilakukan agent. Restart Windows diperlukan; tidak otomatis.') + '\nJika offline, menunggu maksimal 24 jam.')) return;
  const button = document.getElementById('endpointPolicySend'); button.disabled = true;
  try {
    const result = await api('/api/endpoint/lock-policy', {method:'POST', body:JSON.stringify({device,mode,seconds:minutes*60})});
    document.getElementById('endpointPolicyMessage').textContent = result && result.status === 'queued' ? 'Job tersimpan. Gunakan Refresh status untuk melihat hasil.' : result && result.error || 'Respons tidak diketahui; refresh sebelum mencoba ulang.';
    await refreshEndpointDevice();
  } catch (_) { document.getElementById('endpointPolicyMessage').textContent = 'Koneksi gagal; refresh status sebelum mengirim ulang.'; }
}