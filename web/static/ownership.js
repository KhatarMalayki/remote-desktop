// Asset holder changes are always submitted through the audited server workflow.
var responsibilityNotice = 'Aset perusahaan merupakan tanggung jawab pemegang yang tercatat. Sebelum serah terima, pastikan kondisi, kelengkapan, dan data pekerjaan telah diperiksa. Tanggung jawab berpindah setelah switch disetujui. Segera laporkan kehilangan atau kerusakan kepada ADH.';

function isAssetUser() { return currentUser && (currentUser.role === 'user' || currentUser.role === 'spv'); }
function mayReviewSwitch() { return currentUser && ['adh', 'admin', 'ga_pusat'].includes(currentUser.role); }
var handoverDraft = null;

async function requestAssetSwitch(type, id) {
  var asset = type === 'manual' ? manualAssets.find(function(a) { return a.id === id; }) : devices.find(function(a) { return a.id === id; });
  if (!asset) return;
  var isAssignment = !asset.owner_username;
  var branch=asset.branch||asset.group||'Pusat';
  var users=await api('/api/assets/holder-options?branch='+encodeURIComponent(branch));
  if(!Array.isArray(users)){showToast((users&&users.error)||'Daftar PIC tidak dapat dimuat');return}
  if(users.length === 0){
    showToast('Belum ada akun PIC (ADH / SPV) di lokasi ' + branch + '. Silakan buat di menu Akun Cabang.');
    return;
  }
  handoverDraft={type:type,id:id,asset:asset,isAssignment:isAssignment};
  document.getElementById('handoverTitle').textContent=isAssignment?'Tetapkan PIC Awal':'Serah Terima / Switch PIC';
  document.getElementById('handoverAssetName').textContent=(asset.name||asset.hostname)+' · '+branch;
  document.getElementById('handoverCurrentOwner').textContent=asset.owner_username||'Belum ditugaskan';
  document.getElementById('handoverAssignedTo').value = asset.assigned_to || '';
  var select=document.getElementById('handoverTarget');select.innerHTML='<option value="">Pilih akun PIC...</option>'+users.map(function(u){
    var roleLabel = '';
    if(u.role === 'adh') roleLabel = ' (ADH Cabang)';
    else if(u.role === 'spv') roleLabel = ' (SPV Dept)';
    else if(u.role === 'ga_pusat') roleLabel = ' (GA Pusat)';
    var isCur = (u.username === asset.owner_username);
    return '<option value="'+esc(u.username)+'"'+(isCur?' selected':'')+'>'+esc(u.username)+roleLabel+(isCur?' · Tetap':'')+'</option>';
  }).join('');
  document.getElementById('handoverReasonLabel').textContent=isAssignment?'Dasar penetapan PIC awal':'Alasan serah terima';
  document.getElementById('handoverReason').value='';document.getElementById('handoverSwap').value='';document.getElementById('handoverSwapGroup').style.display=isAssignment?'none':'block';document.getElementById('handoverAttachment').value='';document.getElementById('handoverError').style.display='none';
  document.getElementById('handoverSubmit').textContent=mayReviewSwitch()?(isAssignment?'Tetapkan PIC':'Simpan Serah Terima'):'Ajukan Persetujuan';
  document.getElementById('handoverModal').style.display='flex';
}

function closeHandoverModal(){document.getElementById('handoverModal').style.display='none';handoverDraft=null}

async function submitHandover() {
  if(!handoverDraft)return;var target=document.getElementById('handoverTarget').value;var reason=document.getElementById('handoverReason').value.trim();var swapTag=document.getElementById('handoverSwap').value.trim();var attachment=document.getElementById('handoverAttachment').files[0];var error=document.getElementById('handoverError');
  if(!target){error.textContent='Pilih akun PIC tujuan.';error.style.display='block';return}if(reason.length<10){error.textContent='Alasan harus jelas, minimal 10 karakter.';error.style.display='block';return}if(attachment&&attachment.size>10*1024*1024){error.textContent='Lampiran maksimal 10 MB.';error.style.display='block';return}
  var assignedTo = (document.getElementById('handoverAssignedTo')||{}).value || '';
  var button=document.getElementById('handoverSubmit');button.disabled=true;button.textContent='Menyimpan...';
  var result=await api('/api/assets/switch-requests',{method:'POST',body:JSON.stringify({asset_id:handoverDraft.id,asset_type:handoverDraft.type,to_owner:target,assigned_to:assignedTo.trim(),reason:reason,swap_tag:swapTag})});
  if(!result||result.error){error.textContent=(result&&result.error)||'Gagal menyimpan perubahan.';error.style.display='block';button.disabled=false;button.textContent='Coba Lagi';return}
  if (attachment) {
    var uploaded = await uploadHandoverAttachment(result.id, attachment);
    if (!uploaded || uploaded.error) showToast('PIC tersimpan, tetapi lampiran gagal: '+((uploaded&&uploaded.error)||'kesalahan jaringan'));
  }
  if(result.status==='approved'){
    handoverDraft.asset.owner_username=target;
    handoverDraft.asset.assigned_to=assignedTo.trim();
  }
  var wasAssignment=handoverDraft.isAssignment;closeHandoverModal();renderBranchAssets();
  showToast(result.status === 'approved' ? (wasAssignment ? 'PIC awal berhasil ditetapkan' : 'Serah terima berhasil dicatat') : 'Permintaan menunggu persetujuan ADH');
  await loadBranchAssets();
}

async function uploadHandoverAttachment(id, file) {
  var form = new FormData(); form.append('file', file);
  try { var res=await fetch('/api/assets/switch-requests/'+id+'/attachments',{method:'POST',headers:{Authorization:'Bearer '+token},body:form}); return await res.json(); }
  catch(e){ return {error:e.message}; }
}

async function openSwitchHistory() {
  document.getElementById('switchHistoryModal').style.display = 'flex';
  var box = document.getElementById('switchHistoryContent');
  box.textContent = 'Memuat...';
  var rows = await api('/api/assets/switch-requests?limit=100');
  if (!Array.isArray(rows)) { box.textContent = rows && rows.error || 'Tidak dapat memuat riwayat.'; return; }
  box.innerHTML = rows.length ? rows.map(function(r) {
    var actions = r.status === 'pending' && mayReviewSwitch() ? '<button class="btn btn-primary btn-sm" data-review="approve" data-id="'+r.id+'">Setujui</button> <button class="btn btn-danger btn-sm" data-review="reject" data-id="'+r.id+'">Tolak</button>' : '';
    var processLabel = r.operation === 'assignment' ? 'Penetapan PIC awal' : 'Serah terima / switch PIC';
    var status = processLabel + ' · ' + ({pending:'Menunggu approval', approved:'Disetujui', rejected:'Ditolak'}[r.status] || r.status);
    if (r.swap_asset_id) status += ' · Tukar dua aset (' + r.swap_asset_id + ')';
    var files=(r.attachments||[]).map(function(a){return '<button class="btn btn-ghost btn-sm" onclick="downloadAssetAttachment('+a.id+',\''+encodeURIComponent(a.original_name).replace(/'/g,'%27')+'\')">📎 '+esc(a.original_name)+'</button>';}).join(' ');
    return '<div class="switch-entry"><strong>'+esc(r.asset_name)+'</strong> <span class="tag">'+esc(status)+'</span><p>'+esc(r.from_owner || 'Belum ditugaskan')+' → '+esc(r.to_owner)+' · '+esc(r.branch)+'</p><p>Alasan: '+esc(r.reason)+'</p><small>Diajukan '+esc(r.requested_by)+' · '+new Date(r.created_at).toLocaleString()+'</small>'+(r.reviewed_by ? '<p>Diproses '+esc(r.reviewed_by)+' · '+esc(r.review_note)+'</p>' : '')+'<div class="attachment-list">'+files+'</div><div>'+actions+'</div></div>';
  }).join('') : '<p>Belum ada permintaan switch.</p>';
  box.querySelectorAll('[data-review]').forEach(function(button) { button.onclick = function() { reviewAssetSwitch(Number(button.dataset.id), button.dataset.review); }; });
}

async function downloadAssetAttachment(id, encodedName) {
  try { var res=await fetch('/api/assets/attachments/'+id,{headers:{Authorization:'Bearer '+token}}); if(!res.ok){var e=await res.json();alert(e.error||'Lampiran tidak dapat diunduh');return} var blob=await res.blob();var url=URL.createObjectURL(blob);var a=document.createElement('a');a.href=url;a.download=decodeURIComponent(encodedName);a.click();setTimeout(function(){URL.revokeObjectURL(url)},1000); }
  catch(e){alert('Lampiran tidak dapat diunduh: '+e.message)}
}

async function openAssetTimeline(assetId) {
  document.getElementById('activityModal').style.display='flex';
  document.getElementById('activityAssetFilter').value=assetId||'';
  await loadAssetActivities();
}

async function loadAssetActivities() {
  var params=new URLSearchParams(); var asset=document.getElementById('activityAssetFilter').value.trim(); var cat=document.getElementById('activityCategoryFilter').value; var search=document.getElementById('activitySearch').value.trim();var from=document.getElementById('activityFrom').value;var to=document.getElementById('activityTo').value;
  if(asset)params.set('asset_id',asset);if(cat)params.set('category',cat);if(search)params.set('search',search);if(from)params.set('from',from);if(to)params.set('to',to);
  var rows=await api('/api/assets/activities?'+params.toString()); var box=document.getElementById('activityContent');
  if(!Array.isArray(rows)){box.textContent='Gagal memuat aktivitas.';return}
  box.innerHTML=rows.length?rows.map(function(a){return '<div class="switch-entry"><strong>'+esc(a.asset_name||a.asset_id||'Aktivitas')+'</strong> <span class="tag">'+esc(a.category+' · '+a.action)+'</span><p>'+esc(a.detail||'-')+'</p><small>'+esc(a.actor||'sistem')+' · '+esc(a.branch||'-')+' · '+new Date(a.created_at).toLocaleString()+'</small></div>';}).join(''):'<p>Belum ada aktivitas yang cocok.</p>';
}

var relocateDraft = null;

async function relocateAsset(type, id) {
  var asset = type === 'manual' ? manualAssets.find(function(a) { return a.id === id; }) : devices.find(function(a) { return a.id === id; });
  if (!asset) return;
  relocateDraft = { type: type, id: id, asset: asset };
  var currentBranch = asset.branch || asset.group || 'Pusat';
  document.getElementById('relocateAssetName').textContent = (asset.name || asset.hostname) + ' · Lokasi saat ini: ' + currentBranch;
  document.getElementById('relocateAssetId').value = id;
  document.getElementById('relocateAssetType').value = type;
  document.getElementById('relocateToLocation').value = (type === 'manual' ? (asset.location || '') : (asset.local_ip || ''));
  document.getElementById('relocateReason').value = '';
  document.getElementById('relocateError').style.display = 'none';

  var sel = document.getElementById('relocateToBranch');
  sel.innerHTML = '';
  var branches = await api('/api/branches');
  var bList = (Array.isArray(branches) && branches.length) ? branches : [currentBranch];
  bList.forEach(function(b) {
    var opt = document.createElement('option');
    opt.value = b;
    opt.textContent = b;
    if (b === currentBranch) opt.selected = true;
    sel.appendChild(opt);
  });
  document.getElementById('relocateModal').style.display = 'flex';
}

function closeRelocateModal() {
  document.getElementById('relocateModal').style.display = 'none';
  relocateDraft = null;
}

async function submitRelocateModal() {
  if (!relocateDraft) return;
  var toBranch = document.getElementById('relocateToBranch').value.trim();
  var toLoc = document.getElementById('relocateToLocation').value.trim();
  var reason = document.getElementById('relocateReason').value.trim();
  var errEl = document.getElementById('relocateError');

  if (!toBranch) {
    errEl.textContent = 'Pilih lokasi / cabang tujuan.';
    errEl.style.display = 'block';
    return;
  }
  if (reason.length < 10) {
    errEl.textContent = 'Alasan mutasi harus diisi minimal 10 karakter.';
    errEl.style.display = 'block';
    return;
  }

  var btn = document.getElementById('relocateSubmit');
  btn.disabled = true;
  btn.textContent = 'Memproses...';

  var res = await api('/api/assets/relocate', {
    method: 'POST',
    body: JSON.stringify({
      asset_id: relocateDraft.id,
      asset_type: relocateDraft.type,
      to_branch: toBranch,
      to_location: toLoc,
      reason: reason
    })
  });

  btn.disabled = false;
  btn.textContent = 'Simpan Mutasi Lokasi';

  if (!res || res.error) {
    errEl.textContent = (res && res.error) || 'Mutasi gagal.';
    errEl.style.display = 'block';
    return;
  }

  closeRelocateModal();
  showToast('Lokasi aset berhasil dimutasi ke ' + toBranch);
  await loadBranchAssets();
}

var serviceDraft = null;

function openServiceModal(type, id, action) {
  var asset = type === 'manual' ? manualAssets.find(function(a) { return a.id === id; }) : devices.find(function(a) { return a.id === id; });
  if (!asset) return;
  serviceDraft = { type: type, id: id, asset: asset, action: action || 'request' };
  document.getElementById('serviceAssetId').value = id;
  document.getElementById('serviceAssetType').value = type;
  document.getElementById('serviceAction').value = serviceDraft.action;
  document.getElementById('serviceAssetName').textContent = (asset.name || asset.hostname) + ' (' + (asset.branch || asset.group || 'Pusat') + ')';
  document.getElementById('serviceError').style.display = 'none';

  var isComplete = serviceDraft.action === 'complete';
  document.getElementById('serviceModalTitle').textContent = isComplete ? 'Konfirmasi Selesai Servis' : 'Permohonan Servis / Perbaikan';
  document.getElementById('serviceRequestFields').style.display = isComplete ? 'none' : 'block';
  document.getElementById('serviceCompleteFields').style.display = isComplete ? 'block' : 'none';
  document.getElementById('serviceSubmit').textContent = isComplete ? 'Selesaikan & Aktifkan Unit' : 'Kirim Permohonan Servis';

  if (!isComplete) {
    document.getElementById('serviceIssue').value = '';
    document.getElementById('serviceNotes').value = '';
    document.getElementById('serviceCondition').value = (asset.condition === 'damaged') ? 'damaged' : 'fair';
  } else {
    document.getElementById('serviceCompleteNotes').value = '';
  }

  document.getElementById('serviceModal').style.display = 'flex';
}

function closeServiceModal() {
  document.getElementById('serviceModal').style.display = 'none';
  serviceDraft = null;
}

async function submitServiceModal() {
  if (!serviceDraft) return;
  var isComplete = serviceDraft.action === 'complete';
  var errEl = document.getElementById('serviceError');

  var payload = {
    asset_id: serviceDraft.id,
    asset_type: serviceDraft.type,
    action: serviceDraft.action
  };

  if (!isComplete) {
    var issue = document.getElementById('serviceIssue').value.trim();
    var cond = document.getElementById('serviceCondition').value;
    var notes = document.getElementById('serviceNotes').value.trim();
    if (issue.length < 5) {
      errEl.textContent = 'Jelaskan keluhan / kendala kerusakan minimal 5 karakter.';
      errEl.style.display = 'block';
      return;
    }
    payload.issue = issue;
    payload.condition = cond;
    payload.notes = notes;
  } else {
    var cNotes = document.getElementById('serviceCompleteNotes').value.trim();
    if (cNotes.length < 5) {
      errEl.textContent = 'Masukkan catatan hasil perbaikan minimal 5 karakter.';
      errEl.style.display = 'block';
      return;
    }
    payload.notes = cNotes;
  }

  var btn = document.getElementById('serviceSubmit');
  btn.disabled = true;
  btn.textContent = 'Menyimpan...';

  var res = await api('/api/assets/service', {
    method: 'POST',
    body: JSON.stringify(payload)
  });

  btn.disabled = false;
  btn.textContent = isComplete ? 'Selesaikan & Aktifkan Unit' : 'Kirim Permohonan Servis';

  if (!res || res.error) {
    errEl.textContent = (res && res.error) || 'Gagal memproses servis.';
    errEl.style.display = 'block';
    return;
  }

  closeServiceModal();
  showToast(isComplete ? 'Servis selesai, unit kembali aktif normal' : 'Permohonan servis dicatat');
  await loadBranchAssets();
}

async function reviewAssetSwitch(id, action) {
  var note = prompt(action === 'approve' ? 'Catatan persetujuan / hasil pemeriksaan serah terima:' : 'Alasan penolakan:');
  if (note === null) return;
  if (!confirm(responsibilityNotice + '\n\n' + (action === 'approve' ? 'Setujui switch dan pindahkan tanggung jawab?' : 'Tolak permintaan switch ini?'))) return;
  var result = await api('/api/assets/switch-requests/'+id+'/'+action, {method:'POST', body:JSON.stringify({note:note.trim()})});
  if (!result || result.error) { alert(result && result.error || 'Gagal memproses approval'); return; }
  await openSwitchHistory();
  await loadBranchAssets();
}

function ownershipUI() {
  if (!isAssetUser()) return;
  document.getElementById('branchBannerTitle').textContent = 'Aset Saya';
  document.getElementById('branchBannerSubtitle').textContent = responsibilityNotice;
  document.getElementById('btnAddManualAsset').style.display = 'none';
  document.querySelectorAll('[onclick="openDownloadAgentModal()"], .nav-item[data-page="remote"], .nav-item[data-page="dashboard"]').forEach(function(el) { el.style.display = 'none'; });
  document.getElementById('branchSelectFilter').style.display = 'none';
  showPage('branch-assets');
}
