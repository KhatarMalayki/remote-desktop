let vpnPilotState = { ready: false, detail: 'Memuat...', network: '', sessions: {} };
let vpnSelectedDevice = null;
let vpnReturnFocus = null;
let vpnPollTimer = null;
let vpnPolicyLoadedDevice = null;
let vpnMasterDevices = [];
let vpnMasterSelectedDevice = null;
let vpnMasterDevice = null;
let vpnMasterOffset = 0;
let vpnMasterLoadID = 0;

function getVPNDevice(id) {
  return vpnMasterDevice && vpnMasterDevice.id === id ? vpnMasterDevice : vpnMasterDevices.find(device => device.id === id) || (devices || []).find(device => device.id === id);
}

function syncVPNPolling() {
  const visible = currentUser && currentUser.role === 'admin' && ((typeof currentPage !== 'undefined' && currentPage === 'vpn') || document.getElementById('vpnModal').style.display === 'flex');
  if (visible && !vpnPollTimer) vpnPollTimer = setInterval(() => { if (!document.hidden) refreshVPNState(); }, 4000);
  if (!visible && vpnPollTimer) { clearInterval(vpnPollTimer); vpnPollTimer = null; }
}

async function openVPNMaster(deviceId) {
  if (!currentUser || currentUser.role !== 'admin') { showToast('Master VPN hanya untuk admin'); return; }
  closeVPNModal();
  await showPage('vpn');
  if (deviceId) await selectVPNMasterDevice(deviceId);
}

async function loadVPNMaster(offset = vpnMasterOffset) {
  if (!currentUser || currentUser.role !== 'admin') return;
  const request = ++vpnMasterLoadID;
  vpnMasterOffset = Math.max(0, offset);
  document.getElementById('vpnMasterPrev').disabled = true;
  document.getElementById('vpnMasterNext').disabled = true;
  document.getElementById('vpnMasterListStatus').textContent = 'Memuat perangkat...';
  const search = document.getElementById('vpnMasterSearch').value || '';
  const data = await api('/api/devices?search=' + encodeURIComponent(search) + '&offset=' + vpnMasterOffset + '&limit=25');
  if (request !== vpnMasterLoadID || currentPage !== 'vpn') return;
  if (!data || !Array.isArray(data.devices)) {
    vpnMasterDevices = [];
    document.getElementById('vpnMasterRows').replaceChildren();
    document.getElementById('vpnMasterListStatus').textContent = 'Daftar perangkat gagal dimuat. Coba Refresh.';
    return;
  }
  vpnMasterDevices = data.devices;
  if (vpnMasterDevice) vpnMasterDevice = vpnMasterDevices.find(device => device.id === vpnMasterSelectedDevice) || vpnMasterDevice;
  document.getElementById('vpnMasterListStatus').textContent = vpnMasterDevices.length ? (vpnMasterOffset + 1) + '–' + (vpnMasterOffset + vpnMasterDevices.length) + ' dari ' + data.total + ' perangkat' : 'Tidak ada perangkat yang cocok.';
  document.getElementById('vpnMasterPrev').disabled = vpnMasterOffset === 0;
  document.getElementById('vpnMasterNext').disabled = vpnMasterOffset + vpnMasterDevices.length >= data.total;
  await refreshVPNState();
  if (request !== vpnMasterLoadID || currentPage !== 'vpn') return;
  if (!vpnMasterSelectedDevice && vpnMasterDevices.length) await selectVPNMasterDevice(vpnMasterDevices[0].id, false);
}

async function selectVPNMasterDevice(id, focus = true) {
  if (!currentUser || currentUser.role !== 'admin') return;
  const known = getVPNDevice(id);
  vpnMasterSelectedDevice = id;
  vpnMasterDevice = null;
  vpnPolicyLoadedDevice = null;
  document.getElementById('vpnMasterSettings').hidden = true;
  document.getElementById('vpnMasterSelectedName').textContent = 'Memuat perangkat...';
  const device = known || await api('/api/devices/' + encodeURIComponent(id));
  if (vpnMasterSelectedDevice !== id) return;
  if (!device || device.id !== id) {
    document.getElementById('vpnMasterSelectedName').textContent = 'Perangkat tidak tersedia';
    document.getElementById('vpnMasterSelectedInfo').textContent = 'Pilih perangkat lain atau coba lagi.';
    return;
  }
  vpnMasterDevice = device;
  resetVPNLANForm();
  renderVPNMaster();
  await loadVPNSelfPolicy();
  if (focus && vpnMasterSelectedDevice === id) document.getElementById('vpnMasterSelectedName').focus();
}

function renderVPNMaster() {
  document.getElementById('vpnMasterHubStatus').textContent = vpnPilotState.unavailable ? 'Status hub belum diketahui' : vpnPilotState.ready ? 'Hub VPN siap' : 'Hub VPN nonaktif';
  document.getElementById('vpnMasterHubDetail').textContent = (vpnPilotState.detail || '') + (vpnPilotState.network ? ' Subnet: ' + vpnPilotState.network : '');
  const rows = document.getElementById('vpnMasterRows');
  const active = document.activeElement;
  let restoreFocus;
  rows.replaceChildren();
  vpnMasterDevices.forEach(device => {
    const row = document.createElement('tr');
    row.className = device.id === vpnMasterSelectedDevice ? 'vpn-master-selected' : '';
    const identity = document.createElement('td');
    const title = document.createElement('strong');
    title.textContent = device.hostname || device.id;
    const meta = document.createElement('small');
    meta.textContent = device.id + ' · ' + (device.online ? 'Online' : 'Offline') + ' · ' + (device.os || '');
    identity.append(title, meta);
    const status = document.createElement('td');
    const session = (vpnPilotState.sessions || {})[device.id] || {state:'disconnected'};
    status.textContent = vpnPilotState.unavailable ? 'Belum diketahui' : !device.online ? 'Agent offline' : session.state.toUpperCase();
    const address = document.createElement('small');
    address.textContent = [session.address, session.advertise_lan ? 'Gateway ' + session.advertise_lan : (session.routes || []).join(', ')].filter(Boolean).join(' · ');
    status.append(address);
    const actions = document.createElement('td');
    const manage = document.createElement('button');
    manage.type = 'button'; manage.className = 'btn btn-ghost btn-sm'; manage.textContent = 'Kelola';
    manage.onclick = () => selectVPNMasterDevice(device.id);
    manage.vpnDeviceID = device.id;
    const connect = document.createElement('button');
    connect.type = 'button'; connect.className = 'btn btn-ghost btn-sm'; connect.textContent = 'Koneksi';
    connect.onclick = () => openVPNModal(device.id);
    connect.vpnDeviceID = device.id;
    if (active && active.vpnDeviceID === device.id) restoreFocus = active.textContent === 'Kelola' ? manage : connect;
    actions.append(manage, connect);
    row.append(identity, status, actions);
    rows.append(row);
  });
  if (restoreFocus) restoreFocus.focus({preventScroll:true});
  document.getElementById('vpnMasterSettings').hidden = !vpnMasterDevice;
  if (vpnMasterDevice) {
    document.getElementById('vpnMasterSelectedName').textContent = vpnMasterDevice.hostname || vpnMasterDevice.id;
    document.getElementById('vpnMasterSelectedInfo').textContent = vpnMasterDevice.id + ' · ' + (vpnMasterDevice.online ? 'Online' : 'Offline') + ' · ' + (vpnMasterDevice.os || '');
  }
  renderVPNLANControls();
}

async function openVPNModal(deviceId) {
  if (!currentUser || currentUser.role !== 'admin') {
    showToast('Koneksi VPN hanya dapat dikelola admin');
    return;
  }
  vpnReturnFocus = document.activeElement;
  vpnSelectedDevice = deviceId || (devices && devices[0] ? devices[0].id : null);
  document.getElementById('vpnModal').style.display = 'flex';
  updateVPNDevicePicker(vpnSelectedDevice);
  vpnPilotState = {ready: false, unavailable: true, detail: 'Memuat status hub...', sessions: {}};
  renderVPNModal();
  document.getElementById('vpnDevicePicker').focus();
  syncVPNPolling();
  await refreshVPNState();
}

function closeVPNModal() {
  document.getElementById('vpnModal').style.display = 'none';
  syncVPNPolling();
  if (vpnReturnFocus && vpnReturnFocus.isConnected === false && vpnReturnFocus.vpnDeviceID) {
    const previous = vpnReturnFocus;
    vpnReturnFocus = [...document.getElementById('vpnMasterRows').querySelectorAll('button')].find(button => button.vpnDeviceID === previous.vpnDeviceID && button.textContent === previous.textContent) || document.getElementById('vpnMasterSelectedName');
  }
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
  const list = [...new Map([...(devices || []), ...vpnMasterDevices, ...(vpnMasterDevice ? [vpnMasterDevice] : [])].map(device => [device.id, device])).values()];
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
  renderVPNModal();
}

async function loadVPNSelfPolicy() {
  if (!currentUser || currentUser.role !== 'admin') return;
  const device = vpnMasterSelectedDevice;
  vpnPolicyLoadedDevice = null;
  for (const id of ['vpnSelfEnabled', 'vpnSelfRoute', 'btnVPNSelfSave']) document.getElementById(id).disabled = true;
  document.getElementById('vpnSelfEnabled').checked = false;
  document.getElementById('vpnSelfRoute').value = '';
  document.getElementById('vpnSelfStatus').textContent = 'Memuat policy...';
  if (!device) return;
  try {
    const policy = await api('/api/vpn/self-service?device=' + encodeURIComponent(device));
    if (device !== vpnMasterSelectedDevice) return;
    if (!policy || typeof policy.enabled !== 'boolean') throw new Error('Policy tidak dapat dibaca');
    document.getElementById('vpnSelfEnabled').checked = policy.enabled;
    document.getElementById('vpnSelfRoute').value = policy.route_lan || '';
    document.getElementById('vpnSelfStatus').textContent = policy.enabled ? 'Connect mandiri diizinkan admin.' : 'Connect mandiri ditolak.';
    vpnPolicyLoadedDevice = device;
    for (const id of ['vpnSelfEnabled', 'vpnSelfRoute', 'btnVPNSelfSave']) document.getElementById(id).disabled = false;
  } catch (error) {
    if (device === vpnMasterSelectedDevice) document.getElementById('vpnSelfStatus').textContent = 'Gagal memuat policy; penyimpanan dinonaktifkan.';
  }
}

async function saveVPNSelfPolicy() {
  const device = vpnMasterSelectedDevice;
  if (!currentUser || currentUser.role !== 'admin' || !device || vpnPolicyLoadedDevice !== device) return;
  const policy = {enabled: document.getElementById('vpnSelfEnabled').checked, route_lan: document.getElementById('vpnSelfRoute').value.trim()};
  if (!await appConfirm('Simpan policy untuk ' + device + '? Perubahan memutus sesi mandiri aktif. ' + (policy.enabled ? 'Semua pengguna lokal dapat connect ke jaringan VPN pilot; anggota VPN dapat saling mengakses.' : 'Pengguna tidak dapat connect mandiri.'))) return;
  if (device !== vpnMasterSelectedDevice) return;
  document.getElementById('btnVPNSelfSave').disabled = true;
  try {
    const result = await api('/api/vpn/self-service?device=' + encodeURIComponent(device), {method: 'PUT', body: JSON.stringify(policy)});
    if (!result || typeof result.enabled !== 'boolean') throw new Error('Policy gagal disimpan');
    showToast('Policy VPN disimpan.');
    if (device === vpnMasterSelectedDevice) await loadVPNSelfPolicy();
  } catch (error) {
    showToast('Policy VPN gagal disimpan.');
    if (device === vpnMasterSelectedDevice) document.getElementById('btnVPNSelfSave').disabled = false;
  }
}

function resetVPNLANForm() {
  document.getElementById('vpnAdvertiseLAN').checked = false;
  document.getElementById('vpnLANSubnet').value = '';
  const select = document.getElementById('vpnLANClients');
  select.replaceChildren();
  vpnMasterDevices.filter(device => device.id !== vpnMasterSelectedDevice && device.online && device.os === 'windows').forEach(device => {
    const option = document.createElement('option');
    option.value = device.id; option.textContent = device.hostname || device.id; select.append(option);
  });
  document.getElementById('vpnLANClient').value = '';
}

function renderVPNLANControls() {
  const session = (vpnPilotState.sessions || {})[vpnMasterSelectedDevice];
  const active = session && !['disconnected', 'error'].includes(session.state);
  const toggle = document.getElementById('vpnAdvertiseLAN');
  const subnet = document.getElementById('vpnLANSubnet');
  const client = document.getElementById('vpnLANClient');
  if (active) {
    toggle.checked = !!session.advertise_lan;
    subnet.value = session.advertise_lan || '';
    client.value = (session.lan_clients || [])[0] || '';
  }
  toggle.disabled = !vpnMasterDevice || vpnMasterDevice.os !== 'windows' || !!active || !!vpnPilotState.unavailable;
  subnet.disabled = toggle.disabled || !toggle.checked;
  client.disabled = subnet.disabled;
  document.getElementById('btnVPNGatewayConnect').disabled = !vpnPilotState.ready || !vpnMasterDevice || !vpnMasterDevice.online || toggle.disabled || !toggle.checked;
  document.getElementById('btnVPNGatewayDisconnect').disabled = !session || !session.advertise_lan || !active || session.state === 'disconnecting';
  document.getElementById('vpnLANStatus').textContent = vpnPilotState.unavailable || session && session.state === 'error' ? 'Status/cleanup LAN belum terkonfirmasi; periksa detail koneksi.' : active && session.advertise_lan ? 'Gateway LAN: ' + session.advertise_lan + ' — ' + session.state : active && (session.routes || []).length ? 'Akses LAN: ' + session.routes.join(', ') + ' — ' + session.state : 'Advertise LAN OFF';
}

async function refreshVPNState() {
  if (!currentUser || currentUser.role !== 'admin') return;
  try {
    const data = await api('/api/vpn/pilot');
    if (!data || data.error || typeof data.ready !== 'boolean') throw new Error(data && data.error || 'Gagal memuat status VPN. Coba Refresh.');
    vpnPilotState = data;
  } catch (e) {
    vpnPilotState = {ready: false, unavailable: true, detail: e.message || 'Gagal memuat status VPN', sessions: {}};
  }
  renderVPNModal();
  renderVPNMaster();
}

function renderVPNModal() {
  const device = getVPNDevice(vpnSelectedDevice);
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
}

async function triggerVPN(operation, fromMaster = false) {
  const device = fromMaster ? vpnMasterSelectedDevice : vpnSelectedDevice;
  if (!device || !currentUser || currentUser.role !== 'admin' || !['connect','disconnect'].includes(operation)) return;
  const request = {device, operation};
  if (operation === 'connect') {
    let message = 'Aktifkan VPN pilot 15 menit? DNS dan default route tidak diubah.';
    if (fromMaster) {
      if (!document.getElementById('vpnAdvertiseLAN').checked) { showToast('Aktifkan opsi Advertise LAN dahulu.'); return; }
      request.advertise_lan = document.getElementById('vpnLANSubnet').value.trim();
      request.lan_clients = [document.getElementById('vpnLANClient').value.trim()];
      if (!request.advertise_lan || !request.lan_clients[0] || request.lan_clients[0] === device) {showToast('Isi subnet LAN dan pilih satu client lain.'); return;}
      const client = getVPNDevice(request.lan_clients[0]);
      message += '\nGateway: ' + device + '\nBagikan LAN ' + request.advertise_lan + ' hanya kepada ' + (client && client.hostname || request.lan_clients[0]) + '. Forwarding dan NAT Windows akan diaktifkan selama sesi. Client ikut diputus saat gateway berhenti.';
    } else {
      const gateway = Object.values(vpnPilotState.sessions || {}).find(session => session.advertise_lan && (session.lan_clients || []).includes(device) && !['disconnected', 'error'].includes(session.state));
      if (gateway) {request.route_lan = gateway.advertise_lan; message += '\nAkses LAN yang diizinkan: ' + request.route_lan + '.';}
    }
    if (!await appConfirm(message)) return;
  }
  if (operation === 'disconnect' && ((vpnPilotState.sessions || {})[device] || {}).advertise_lan && !await appConfirm('Putuskan gateway? Akses LAN dicabut; client LAN ikut diputus.')) return;
  if (device !== (fromMaster ? vpnMasterSelectedDevice : vpnSelectedDevice)) return;
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
