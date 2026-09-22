// Asset holder changes are always submitted through the audited server workflow.
var responsibilityNotice = 'Aset perusahaan merupakan tanggung jawab pemegang yang tercatat. Sebelum serah terima, pastikan kondisi, kelengkapan, dan data pekerjaan telah diperiksa. Tanggung jawab berpindah setelah switch disetujui. Segera laporkan kehilangan atau kerusakan kepada ADH.';

function isAssetUser() { return currentUser && currentUser.role === 'user'; }
function mayReviewSwitch() { return currentUser && ['adh', 'admin', 'ga_pusat'].includes(currentUser.role); }
var handoverDraft = null;

async function requestAssetSwitch(type, id) {
  var asset = type === 'manual' ? manualAssets.find(function(a) { return a.id === id; }) : devices.find(function(a) { return a.id === id; });
  if (!asset) return;
  var isAssignment = !asset.owner_username;
  var branch=asset.branch||asset.group||'Pusat';
  var users=await api('/api/assets/holder-options?branch='+encodeURIComponent(branch));
  if(!Array.isArray(users)){showToast((users&&users.error)||'Daftar PIC tidak dapat dimuat');return}
  handoverDraft={type:type,id:id,asset:asset,isAssignment:isAssignment};
  document.getElementById('handoverTitle').textContent=isAssignment?'Tetapkan PIC Awal':'Serah Terima / Switch PIC';
  document.getElementById('handoverAssetName').textContent=(asset.name||asset.hostname)+' · '+branch;
  document.getElementById('handoverCurrentOwner').textContent=asset.owner_username||'Belum ditugaskan';
  var select=document.getElementById('handoverTarget');select.innerHTML='<option value="">Pilih akun PIC...</option>'+users.filter(function(u){return u.username!==asset.owner_username}).map(function(u){return '<option value="'+esc(u.username)+'">'+esc(u.username)+'</option>'}).join('');
  document.getElementById('handoverReasonLabel').textContent=isAssignment?'Dasar penetapan PIC awal':'Alasan serah terima';
  document.getElementById('handoverReason').value='';document.getElementById('handoverSwap').value='';document.getElementById('handoverSwapGroup').style.display=isAssignment?'none':'block';document.getElementById('handoverAttachment').value='';document.getElementById('handoverError').style.display='none';
  document.getElementById('handoverSubmit').textContent=mayReviewSwitch()?(isAssignment?'Tetapkan PIC':'Simpan Serah Terima'):'Ajukan Persetujuan';
  document.getElementById('handoverModal').style.display='flex';
}

function closeHandoverModal(){document.getElementById('handoverModal').style.display='none';handoverDraft=null}

async function submitHandover() {
  if(!handoverDraft)return;var target=document.getElementById('handoverTarget').value;var reason=document.getElementById('handoverReason').value.trim();var swapTag=document.getElementById('handoverSwap').value.trim();var attachment=document.getElementById('handoverAttachment').files[0];var error=document.getElementById('handoverError');
  if(!target){error.textContent='Pilih akun PIC tujuan.';error.style.display='block';return}if(reason.length<10){error.textContent='Alasan harus jelas, minimal 10 karakter.';error.style.display='block';return}if(attachment&&attachment.size>10*1024*1024){error.textContent='Lampiran maksimal 10 MB.';error.style.display='block';return}
  var button=document.getElementById('handoverSubmit');button.disabled=true;button.textContent='Menyimpan...';
  var result=await api('/api/assets/switch-requests',{method:'POST',body:JSON.stringify({asset_id:handoverDraft.id,asset_type:handoverDraft.type,to_owner:target,reason:reason,swap_tag:swapTag})});
  if(!result||result.error){error.textContent=(result&&result.error)||'Gagal menyimpan perubahan.';error.style.display='block';button.disabled=false;button.textContent='Coba Lagi';return}
  if (attachment) {
    var uploaded = await uploadHandoverAttachment(result.id, attachment);
    if (!uploaded || uploaded.error) showToast('PIC tersimpan, tetapi lampiran gagal: '+((uploaded&&uploaded.error)||'kesalahan jaringan'));
  }
  if(result.status==='approved')handoverDraft.asset.owner_username=target;
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

async function relocateAsset(type, id) {
  var asset=type==='manual'?manualAssets.find(function(a){return a.id===id}):devices.find(function(a){return a.id===id});if(!asset)return;
  var current=asset.branch||asset.group||'';var target=prompt('Lokasi/cabang tujuan (harus sama persis dengan lokasi terdaftar):',current);if(!target)return;
  var point=prompt('Titik lokasi baru (ruang, meja, lantai):',type==='manual'?(asset.location||''):'');if(point===null)return;
  var reason=prompt('Alasan mutasi lokasi (minimal 10 karakter):');if(!reason||reason.trim().length<10){alert('Alasan minimal 10 karakter.');return}
  if(!confirm('Pindahkan '+(asset.name||asset.hostname)+' dari '+current+' ke '+target.trim()+'?'))return;
  var res=await api('/api/assets/relocate',{method:'POST',body:JSON.stringify({asset_id:id,asset_type:type,to_branch:target.trim(),to_location:point.trim(),reason:reason.trim()})});if(!res||res.error){alert((res&&res.error)||'Mutasi gagal');return}showToast('Lokasi aset berhasil diperbarui');await loadBranchAssets();
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
