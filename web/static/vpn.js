let vpnPilotState = { ready: false, detail: 'Memuat...', network: '', sessions: {} };
let vpnSelectedDevice = null;
let vpnReturnFocus = null;
let vpnPollTimer = null;
let vpnPolicyLoadedDevice = null;

async function openVPNModal(deviceId) {
  if (!currentUser || currentUser.role !== 'admin') {
    showToast('Hanya admin yang dapat menguji VPN pilot');
    return;
  }
  vpnReturnFocus = document.activeElement;
  vpnSelectedDevice = deviceId || (devices && devices[0] ? devices[0].id : null);
  document.getElementById('vpnModal').style.display = 'flex';
  updateVPNDevicePicker(vpnSelectedDevice);
  resetVPNLANForm();
  vpnPilotState = {ready: false, unavailable: true, detail: 'Memuat status hub...', sessions: {}};
  renderVPNModal();
  document.getElementById('vpnDevicePicker').focus();
  if (vpnPollTimer) clearInterval(vpnPollTimer);
  vpnPollTimer = setInterval(refreshVPNState, 4000);
  await refreshVPNState();
  await loadVPNSelfPolicy();
}

function closeVPNModal() {
  document.getElementById('vpnModal').style.display = 'none';
  if (vpnPollTimer) { clearInterval(vpnPollTimer); vpnPollTimer = null; }
  if (vpnReturnFocus && typeof vpnReturnFocus.focus === 'function') vpnReturnFocus.focus();
}

function handleVPNDialogKeydown(event) {
  if (event.key === 'Escape') { event.preventDefault(); closeVPNModal(); return; }
  if (event.key !== 'Tab') return;
  const controls = event.currentTarget.querySelectorAll('button:not(:disabled), select:not(:disabled), input:not(:disabled)');
  const first = controls[0], last = controls[controls.length - 1];
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
}

function updateVPNDevicePicker(activeId) {
  const select = document.getElementById('vpnDevicePicker');
  if (!select) return;
  select.replaceChildren();
  const list = (typeof devices !== 'undefined' && Array.isArray(devices)) ? devices : [];
  list.forEach(d => {
    const opt = document.createElement('option');
    opt.value = d.id;
    opt.textContent = (d.hostname || d.id) + (d.online ? ' (Online)' : ' (Offline)') + ' [' + (d.os || '') + ']';
    if (d.id === activeId) opt.selected = true;
    select.append(opt);
  });
}

function onVPNDevicePickerChange(id) {
  vpnSelectedDevice = id;
  resetVPNLANForm();
  renderVPNModal();
  loadVPNSelfPolicy();
}

async function loadVPNSelfPolicy() {
  const device = vpnSelectedDevice;
  vpnPolicyLoadedDevice = null;
  for (const id of ['vpnSelfEnabled', 'vpnSelfRoute', 'btnVPNSelfSave']) document.getElementById(id).disabled = true;
  document.getElementById('vpnSelfEnabled').checked = false;
  document.getElementById('vpnSelfRoute').value = '';
  document.getElementById('vpnSelfStatus').textContent = 'Memuat policy...';
  if (!device) return;
  try {
    const policy = await api('/api/vpn/self-service?device=' + encodeURIComponent(device));
    if (device !== vpnSelectedDevice) return;
    if (!policy || typeof policy.enabled !== 'boolean') throw new Error('Policy tidak dapat dibaca');
    document.getElementById('vpnSelfEnabled').checked = policy.enabled;
    document.getElementById('vpnSelfRoute').value = policy.route_lan || '';
    document.getElementById('vpnSelfStatus').textContent = policy.enabled ? 'Connect mandiri diizinkan admin.' : 'Connect mandiri ditolak.';
    vpnPolicyLoadedDevice = device;
    for (const id of ['vpnSelfEnabled', 'vpnSelfRoute', 'btnVPNSelfSave']) document.getElementById(id).disabled = false;
  } catch (error) {
    if (device === vpnSelectedDevice) document.getElementById('vpnSelfStatus').textContent = 'Gagal memuat policy; penyimpanan dinonaktifkan.';
  }
}

async function saveVPNSelfPolicy() {
  const device = vpnSelectedDevice;
  if (!currentUser || currentUser.role !== 'admin' || !device || vpnPolicyLoadedDevice !== device) return;
  const policy = {enabled: document.getElementById('vpnSelfEnabled').checked, route_lan: document.getElementById('vpnSelfRoute').value.trim()};
  if (!await appConfirm('Simpan policy untuk ' + device + '? Perubahan memutus sesi mandiri aktif. ' + (policy.enabled ? 'Semua pengguna lokal dapat connect ke jaringan VPN pilot; anggota VPN dapat saling mengakses.' : 'Pengguna tidak dapat connect mandiri.'))) return;
  if (device !== vpnSelectedDevice) return;
  document.getElementById('btnVPNSelfSave').disabled = true;
  try {
    const result = await api('/api/vpn/self-service?device=' + encodeURIComponent(device), {method: 'PUT', body: JSON.stringify(policy)});
    if (!result || typeof result.enabled !== 'boolean') throw new Error('Policy gagal disimpan');
    showToast('Policy VPN disimpan.');
    if (device === vpnSelectedDevice) await loadVPNSelfPolicy();
  } catch (error) {
    showToast('Policy VPN gagal disimpan.');
    if (device === vpnSelectedDevice) document.getElementById('btnVPNSelfSave').disabled = false;
  }
}

function resetVPNLANForm() {
  document.getElementById('vpnAdvertiseLAN').checked = false;
  document.getElementById('vpnLANSubnet').value = '';
  const select = document.getElementById('vpnLANClients');
  select.replaceChildren();
  (devices || []).filter(device => device.id !== vpnSelectedDevice && device.online && device.os === 'windows').forEach(device => {
    const option = document.createElement('option');
    option.value = device.id; option.textContent = device.hostname || device.id; select.append(option);
  });
  document.getElementById('vpnLANClient').value = '';
}

function renderVPNLANControls() {
  const session = (vpnPilotState.sessions || {})[vpnSelectedDevice];
  const active = session && !['disconnected', 'error'].includes(session.state);
  const toggle = document.getElementById('vpnAdvertiseLAN');
  const subnet = document.getElementById('vpnLANSubnet');
  const client = document.getElementById('vpnLANClient');
  if (active) {
    toggle.checked = !!session.advertise_lan;
    subnet.value = session.advertise_lan || '';
    client.value = (session.lan_clients || [])[0] || '';
  }
  toggle.disabled = !!active || !!vpnPilotState.unavailable;
  subnet.disabled = toggle.disabled || !toggle.checked;
  client.disabled = subnet.disabled;
  document.getElementById('vpnLANStatus').textContent = vpnPilotState.unavailable || session && session.state === 'error' ? 'Status/cleanup LAN belum terkonfirmasi; periksa detail koneksi.' : active && session.advertise_lan ? 'Gateway LAN: ' + session.advertise_lan + ' — ' + session.state : active && (session.routes || []).length ? 'Akses LAN: ' + session.routes.join(', ') + ' — ' + session.state : 'Advertise LAN OFF';
}

async function refreshVPNState() {
  try {
    const data = await api('/api/vpn/pilot');
    if (!data || data.error || typeof data.ready !== 'boolean') throw new Error(data && data.error || 'Gagal memuat status VPN. Coba Refresh.');
    vpnPilotState = data;
  } catch (e) {
    vpnPilotState = {ready: false, unavailable: true, detail: e.message || 'Gagal memuat status VPN', sessions: {}};
  }
  renderVPNModal();
}

function renderVPNModal() {
  const device = (typeof devices !== 'undefined' && Array.isArray(devices)) ? devices.find(d => d.id === vpnSelectedDevice) : null;
  const session = vpnPilotState.unavailable ? {state: 'unknown', detail: 'Status target belum dapat dipastikan. Coba Refresh; disconnect darurat tersedia di tray Windows target.'} : (vpnPilotState.sessions || {})[vpnSelectedDevice] || { state: 'disconnected', detail: 'VPN nonaktif; internet client memakai jalur biasa' };
  document.getElementById('vpnHubStatus').textContent = vpnPilotState.unavailable ? 'Belum diketahui' : vpnPilotState.ready ? 'Siap' : 'Nonaktif';
  document.getElementById('vpnHubStatus').className = 'badge-status ' + (vpnPilotState.ready ? 'online' : 'offline');
  document.getElementById('vpnHubDetail').textContent = vpnPilotState.detail || '-';
  document.getElementById('vpnNetworkInfo').textContent = vpnPilotState.network || 'Belum diatur';
  const targetInfo = device ? ((device.hostname || device.id) + ' — ' + (device.online ? 'Online' : 'Offline') + ' (' + (device.os || '') + ' ' + (device.version || '') + ')') : 'Pilih perangkat';
  document.getElementById('vpnTargetInfo').textContent = targetInfo;
  document.getElementById('vpnClientStatus').textContent = session.state.toUpperCase();
  document.getElementById('vpnClientDetail').textContent = session.detail || '-';
  document.getElementById('vpnAssignedIP').textContent = vpnPilotState.unavailable ? 'Belum diketahui' : session.address ? (session.address + ' (khusus pilot)') : 'Belum terhubung';
  const btnConnect = document.getElementById('btnVPNConnect');
  const btnDisconnect = document.getElementById('btnVPNDisconnect');
  const isBusy = session.state === 'preparing' || session.state === 'connecting' || session.state === 'disconnecting';
  btnConnect.disabled = !vpnPilotState.ready || !device || !device.online || device.os !== 'windows' || session.state === 'running' || session.state === 'connected' || isBusy;
  btnDisconnect.disabled = !device || session.state === 'disconnected' || session.state === 'disconnecting';
  renderVPNLANControls();
}

async function triggerVPN(operation) {
  if (!vpnSelectedDevice) return;
  const request = {device: vpnSelectedDevice, operation};
  if (operation === 'connect') {
    let message = 'Aktifkan VPN pilot 15 menit? DNS dan default route tidak diubah.';
    if (document.getElementById('vpnAdvertiseLAN').checked) {
      request.advertise_lan = document.getElementById('vpnLANSubnet').value.trim();
      request.lan_clients = [document.getElementById('vpnLANClient').value.trim()];
      if (!request.advertise_lan || !request.lan_clients[0] || request.lan_clients[0] === vpnSelectedDevice) {showToast('Isi subnet LAN dan pilih satu client lain.'); return;}
      const client = (devices || []).find(device => device.id === request.lan_clients[0]);
      message += '\nGateway: ' + vpnSelectedDevice + '\nBagikan LAN ' + request.advertise_lan + ' hanya kepada ' + (client && client.hostname || request.lan_clients[0]) + '. Forwarding dan NAT Windows akan diaktifkan selama sesi. Client ikut diputus saat gateway berhenti.';
    } else {
      const gateway = Object.values(vpnPilotState.sessions || {}).find(session => session.advertise_lan && (session.lan_clients || []).includes(vpnSelectedDevice) && !['disconnected', 'error'].includes(session.state));
      if (gateway) {request.route_lan = gateway.advertise_lan; message += '\nAkses LAN yang diizinkan: ' + request.route_lan + '.';}
    }
    if (!await appConfirm(message)) return;
  }
  if (operation === 'disconnect' && ((vpnPilotState.sessions || {})[vpnSelectedDevice] || {}).advertise_lan && !await appConfirm('Putuskan gateway? Akses LAN dicabut; client LAN ikut diputus.')) return;
  try {
    const data = await api('/api/vpn/pilot', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request)
    });
    if (!data || data.error) {
      showToast(data && data.error || 'Gagal menghubungi server VPN');
    } else {
      showToast(operation === 'connect' ? 'Permintaan connect dikirim; menunggu pengaman lokal' : 'Permintaan disconnect dikirim');
      await refreshVPNState();
    }
  } catch (e) {
    showToast('Gagal menghubungi server');
  }
}
