let vpnPilotState = { ready: false, detail: 'Memuat...', network: '', sessions: {} };
let vpnSelectedDevice = null;
let vpnReturnFocus = null;
let vpnPollTimer = null;

async function openVPNModal(deviceId) {
  if (!currentUser || currentUser.role !== 'admin') {
    showToast('Hanya admin yang dapat menguji VPN pilot');
    return;
  }
  vpnReturnFocus = document.activeElement;
  vpnSelectedDevice = deviceId || (devices && devices[0] ? devices[0].id : null);
  document.getElementById('vpnModal').style.display = 'flex';
  updateVPNDevicePicker(vpnSelectedDevice);
  await refreshVPNState();
  if (vpnPollTimer) clearInterval(vpnPollTimer);
  vpnPollTimer = setInterval(refreshVPNState, 4000);
}

function closeVPNModal() {
  document.getElementById('vpnModal').style.display = 'none';
  if (vpnPollTimer) { clearInterval(vpnPollTimer); vpnPollTimer = null; }
  if (vpnReturnFocus && typeof vpnReturnFocus.focus === 'function') vpnReturnFocus.focus();
}

function updateVPNDevicePicker(activeId) {
  const select = document.getElementById('vpnDevicePicker');
  if (!select) return;
  select.replaceChildren();
  const list = (typeof devices !== 'undefined' && Array.isArray(devices)) ? devices : [];
  list.forEach(d => {
    const opt = document.createElement('option');
    opt.value = d.id;
    opt.textContent = (d.hostname || d.id) + (d.online ? ' (Online)' : ' (Offline)') + ' [' + esc(d.os || '') + ']';
    if (d.id === activeId) opt.selected = true;
    select.append(opt);
  });
}

function onVPNDevicePickerChange(id) {
  vpnSelectedDevice = id;
  renderVPNModal();
}

async function refreshVPNState() {
  try {
    const res = await apiFetch('/api/vpn/pilot');
    if (res.ok) {
      vpnPilotState = await res.json();
      renderVPNModal();
    }
  } catch (e) {}
}

function renderVPNModal() {
  const device = (typeof devices !== 'undefined' && Array.isArray(devices)) ? devices.find(d => d.id === vpnSelectedDevice) : null;
  const session = (vpnPilotState.sessions || {})[vpnSelectedDevice] || { state: 'disconnected', detail: 'VPN nonaktif; internet client memakai jalur biasa' };
  document.getElementById('vpnHubStatus').textContent = vpnPilotState.ready ? 'Siap' : 'Nonaktif';
  document.getElementById('vpnHubStatus').className = 'badge-status ' + (vpnPilotState.ready ? 'online' : 'offline');
  document.getElementById('vpnHubDetail').textContent = vpnPilotState.detail || '-';
  document.getElementById('vpnNetworkInfo').textContent = vpnPilotState.network || 'Belum diatur';
  const targetInfo = device ? (esc(device.hostname || device.id) + ' — ' + (device.online ? 'Online' : 'Offline') + ' (' + esc(device.os || '') + ' ' + esc(device.version || '') + ')') : 'Pilih perangkat';
  document.getElementById('vpnTargetInfo').textContent = targetInfo;
  document.getElementById('vpnClientStatus').textContent = session.state.toUpperCase();
  document.getElementById('vpnClientDetail').textContent = session.detail || '-';
  document.getElementById('vpnAssignedIP').textContent = session.address ? (session.address + ' (khusus pilot)') : 'Belum terhubung';
  const btnConnect = document.getElementById('btnVPNConnect');
  const btnDisconnect = document.getElementById('btnVPNDisconnect');
  const isBusy = session.state === 'preparing' || session.state === 'connecting' || session.state === 'disconnecting';
  btnConnect.disabled = !vpnPilotState.ready || !device || !device.online || device.os !== 'windows' || session.state === 'running' || session.state === 'connected' || isBusy;
  btnDisconnect.disabled = !device || session.state === 'disconnected' || isBusy;
}

async function triggerVPN(operation) {
  if (!vpnSelectedDevice) return;
  if (operation === 'connect' && !await appConfirm('Aktifkan VPN pilot 15 menit pada perangkat ini? Hanya subnet VPN uji yang dilewatkan; DNS dan internet biasa tidak diubah. Disconnect darurat tersedia di tray Windows target.')) return;
  try {
    const res = await apiFetch('/api/vpn/pilot', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ device: vpnSelectedDevice, operation: operation })
    });
    const data = await res.json();
    if (!res.ok) {
      showToast(data.error || 'Operasi VPN ditolak');
    } else {
      showToast(operation === 'connect' ? 'Permintaan connect dikirim; menunggu pengaman lokal' : 'Permintaan disconnect dikirim');
      await refreshVPNState();
    }
  } catch (e) {
    showToast('Gagal menghubungi server');
  }
}
