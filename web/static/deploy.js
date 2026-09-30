function previewRemoteFile(input) {
  const file = input.files && input.files[0];
  document.getElementById('localFilePreview').textContent = file ? file.name + ' — ' + fmtBytes(file.size) : 'Belum ada file dipilih.';
}
async function openFileManager() {
  if (!remoteWS || !supportsDeployment(remoteWS.agentVersion)) return appAlert('Update agent ke 0.2.56 untuk browser folder dan lokasi tujuan.');
  if (!remoteWS || remoteWS.readyState !== WebSocket.OPEN) return;
  document.getElementById('fileManagerModal').style.display = 'flex';
  await browseRemoteFiles();
}
async function browseRemoteFiles() {
  if (remoteFileBusy || !remoteWS || remoteWS.readyState !== WebSocket.OPEN) return;
  const output = document.getElementById('remoteFileEntries');
  remoteFileBusy = true;
  try {
    const result = await remoteFileRequest(remoteWS, {type:'file_list', destination:document.getElementById('remoteDestination').value.trim()});
    if (!result.path || !Array.isArray(result.entries)) throw new Error('Update agent ke 0.2.56 untuk browser folder.');
    document.getElementById('remoteDestination').value = result.path;
    output.replaceChildren();
    const parent = document.createElement('button');
    parent.className = 'btn btn-ghost'; parent.textContent = '.. (folder induk)';
    parent.onclick = () => { document.getElementById('remoteDestination').value = result.parent; browseRemoteFiles(); };
    output.append(parent);
    for (const entry of result.entries) {
      const item = document.createElement(entry.directory ? 'button' : 'div');
      item.textContent = (entry.directory ? '[Folder] ' : '') + entry.name;
      if (entry.directory) { item.className = 'btn btn-ghost'; item.onclick = () => { document.getElementById('remoteDestination').value = result.path.replace(/[\\/]$/, '') + (result.path.includes('\\') ? '\\' : '/') + entry.name; browseRemoteFiles(); }; }
      output.append(item);
    }
    if (result.truncated) output.append('Hanya 500 entri pertama.');
  } catch (error) { output.textContent = error.message; }
  finally { remoteFileBusy = false; }
}
function supportsDeployment(version) {
  const parts = String(version || '').replace(/^v/, '').split('.').map(Number);
  return parts.length === 3 && parts.every(Number.isInteger) && (parts[0] > 0 || parts[1] > 2 || (parts[1] === 2 && parts[2] >= 56));
}
let deploymentDevices = [];
let deploymentPoll = null;
function closeDeployments() {
  document.getElementById('deployModal').style.display = 'none';
  clearTimeout(deploymentPoll);
}
async function openDeployments() {
  if (!currentUser || !['admin','it_support'].includes(currentUser.role)) return appAlert('Hanya admin dan IT Support.');
  document.getElementById('deployModal').style.display = 'flex';
  deploymentDevices = [];
  document.getElementById('deployTargets').textContent = 'Memuat perangkat...';
  for (let offset = 0; offset < 10000; offset += 100) {
    const page = await api('/api/devices?limit=100&offset=' + offset);
    if (!page || !Array.isArray(page.devices)) { document.getElementById('deployTargets').textContent = 'Gagal memuat perangkat'; return; }
    deploymentDevices.push(...page.devices);
    if (!page.devices.length || offset + page.devices.length >= page.total) break;
  }
  renderDeployTargets();
  async function refresh() {
    if (document.getElementById('deployModal').style.display === 'none') return;
    await loadDeployments();
    deploymentPoll = setTimeout(refresh, 5000);
  }
  clearTimeout(deploymentPoll);
  await refresh();
}
function renderDeployTargets() {
  const output = document.getElementById('deployTargets'); output.replaceChildren();
  const filter = document.getElementById('deployBranch').value.toLowerCase();
  for (const device of deploymentDevices.filter(item => item.online && item.os === 'windows' && supportsDeployment(item.version) && (item.branch || item.group || '').toLowerCase().includes(filter))) {
    const label = document.createElement('label'); label.style.display = 'block';
    const checkbox = document.createElement('input'); checkbox.type = 'checkbox'; checkbox.value = device.id;
    label.append(checkbox, ' ' + device.hostname + ' — ' + (device.branch || device.group || '') + ' (v' + device.version + ')'); output.append(label);
  }
}
async function submitDeployment() {
  const installer = document.getElementById('deployInstaller').files[0];
  const targets = Array.from(document.querySelectorAll('#deployTargets input:checked'), item => item.value);
  let args;
  try { args = JSON.parse(document.getElementById('deployArgs').value); if (!Array.isArray(args) || args.some(arg => typeof arg !== 'string')) throw new Error(); }
  catch (_) { return appAlert('Argumen wajib array JSON berisi string.'); }
  if (!installer || installer.size > 100 * 1024 * 1024 || !targets.length || targets.length > 50) return appAlert('Pilih installer <=100 MiB dan 1–50 target.');
  if (!await appConfirm(['Jalankan ' + installer.name + ' pada ' + targets.length + ' perangkat berikut?', ...targets, 'Installer memakai hak akses Agent. Pastikan sumber tepercaya, argumen silent/no-restart sesuai vendor. EXE dapat mengabaikan no-restart.'].join('\n'))) return;
  const button = document.getElementById('deploySubmit'); button.disabled = true;
  try {
    const body = new FormData(); body.append('installer', installer); body.append('targets', JSON.stringify(targets)); body.append('args', JSON.stringify(args));
    const response = await fetch('/api/deployments', {method:'POST', headers:{Authorization:'Bearer '+token}, body});
    const result = await response.json(); if (!response.ok) throw new Error(result.error || 'Deploy ditolak');
    await loadDeployments();
  } catch (error) { await appAlert(error.message + ' Jika koneksi terputus, refresh status sebelum mengirim ulang.'); }
  finally { button.disabled = false; }
}
async function loadDeployments() {
  const result = await api('/api/deployments');
  const output = document.getElementById('deployResults'); output.replaceChildren();
  if (!Array.isArray(result)) { output.textContent = result && result.error || 'Gagal membaca status'; return; }
  for (const job of result) { const row = document.createElement('p'); row.textContent = job.device + ' | ' + job.name + ' | ' + job.status + ' | ' + job.detail; output.append(row); }
}
