let endpointSelectedDevice = null;
let endpointReturnFocus = null;

function endpointBadges(device) {
  const state = device.endpoint || {};
  const stale = !device.online || !state.checked_at || Date.now() / 1000 - state.checked_at > 900;
  const badges = (state.applications || []).map(app => {
    let icon = '⚪';
    let iconColor = '#94a3b8';
    let badgeBg = 'rgba(160,163,177,0.15)';
    let badgeBorder = '1px solid rgba(160,163,177,0.3)';
    let statusText = 'status belum diketahui';

    if (app.status === 'detected') {
      icon = '✔';
      iconColor = '#10b981';
      badgeBg = 'rgba(16,185,129,0.15)';
      badgeBorder = '1px solid rgba(16,185,129,0.35)';
      statusText = 'terpasang';
    } else if (app.status === 'not_detected') {
      icon = '✖';
      iconColor = '#ef4444';
      badgeBg = 'rgba(239,68,68,0.12)';
      badgeBorder = '1px solid rgba(239,68,68,0.25)';
      statusText = 'tidak terpasang';
    }

    const title = esc(app.label + ': ' + statusText + (stale ? ' (data lama/offline)' : ''));
    return '<span style="display:inline-flex;align-items:center;gap:3px;padding:2px 7px;border-radius:4px;font-size:11px;font-weight:600;margin:2px 3px 2px 0;background:' + badgeBg + ';border:' + badgeBorder + '" title="' + title + '">' +
      '<span style="font-weight:bold;color:' + iconColor + '">' + icon + '</span> ' +
      esc(app.label) +
    '</span>';
  }).join('');

  return (badges || '<span style="color:var(--fg2);font-size:11px">-</span>') +
    '<br><button class="btn btn-ghost btn-sm" style="margin-top:4px;padding:2px 8px;font-size:11px" data-device="' + esc(device.id) + '" onclick="openEndpointDevice(this.dataset.device)">🛡️ Kelola Policy</button>';
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
  if (!device || device.error) { document.getElementById('endpointDeviceInfo').textContent = device && device.error || 'Gagal membaca perangkat'; document.getElementById('endpointPolicyControls').style.display = 'none'; return; }
  const state = device.endpoint || {}, report = state.report || {}, policy = state.policy;
  document.getElementById('endpointDeviceTitle').textContent = (device.hostname || device.id) + ' — Kelola Policy Agent';
  const output = document.getElementById('endpointDeviceInfo'); output.replaceChildren();
  function line(text) { const element = document.createElement('p'); element.textContent = text; output.append(element); }
  line('Pemeriksaan terakhir: ' + (state.checked_at ? new Date(state.checked_at * 1000).toLocaleString() : 'belum ada; perlu agent 0.2.58+') + (device.online ? '' : ' — perangkat offline, data historis'));
  line('Cakupan: aplikasi machine-wide dan service Windows. Tidak terdeteksi bukan bukti aplikasi portable/per-user tidak ada. Service berjalan bukan bukti proteksi sehat/tersambung console.');
  for (const app of state.applications || []) {
    const icon = app.status === 'detected' ? '✔ ' : app.status === 'not_detected' ? '✖ ' : '⚪ ';
    line(icon + app.label + ': ' + ({detected:'terdeteksi', not_detected:'tidak terdeteksi dalam cakupan', unknown:'belum diketahui'}[app.status] || 'belum diketahui'));
    for (const match of app.matches || []) line('  ' + match.name + (match.version ? ' v' + match.version : '') + (match.service ? ' — service ' + match.service : ''));
  }
  if (report.detail) line('Inventaris: ' + report.detail);
  line('Lock-screen registry: ' + (state.report && report.status !== 'unsupported' ? (report.lock_present ? report.lock_seconds + ' detik' : 'belum diatur') : 'belum diketahui'));
  if (report.policy_error) line('Pembatasan/pemeriksaan policy: ' + report.policy_error);
  if (report.drift) line('PERHATIAN: pengaturan berubah di luar RemoteDesk. Tidak ditimpa otomatis.');
  if (policy) line('Job terakhir: ' + policy.status + ' — ' + policy.detail);
  const supported = device.os === 'windows' && !isVersionNewer('0.2.58', device.version || '');
  document.getElementById('endpointPolicyControls').style.display = currentUser && currentUser.role === 'admin' ? 'block' : 'none';
  document.getElementById('endpointPolicySend').disabled = !supported || !!(policy && ['pending','delivered'].includes(policy.status));
  if (!supported) line('Policy memerlukan Windows agent >=0.2.58.');
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