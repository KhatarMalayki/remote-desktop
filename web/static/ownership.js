// Asset holder changes are always submitted through the audited server workflow.
var responsibilityNotice = 'Aset perusahaan merupakan tanggung jawab pemegang yang tercatat. Sebelum serah terima, pastikan kondisi, kelengkapan, dan data pekerjaan telah diperiksa. Tanggung jawab berpindah setelah switch disetujui. Segera laporkan kehilangan atau kerusakan kepada ADH.';

function isAssetUser() { return currentUser && currentUser.role === 'user'; }
function mayReviewSwitch() { return currentUser && ['adh', 'admin', 'ga_pusat'].includes(currentUser.role); }

async function requestAssetSwitch(type, id) {
  var asset = type === 'manual' ? manualAssets.find(function(a) { return a.id === id; }) : devices.find(function(a) { return a.id === id; });
  if (!asset) return;
  var target = prompt('Username akun pemegang baru di lokasi yang sama (buat akun User terlebih dahulu):');
  if (!target || !target.trim()) return;
  var swapTag = asset.owner_username ? prompt('Untuk tukar dua aset: masukkan tag inventaris / hostname laptop yang diterima sebagai pengganti. Kosongkan jika hanya memindahkan pemegang.', '') : '';
  if (swapTag === null) return;
  var reason = prompt('Alasan switch / serah terima (minimal 10 karakter):');
  if (!reason || reason.trim().length < 10) { alert('Alasan harus jelas, minimal 10 karakter.'); return; }
  var direct = mayReviewSwitch();
  if (!confirm(responsibilityNotice + '\n\nAset: ' + (asset.name || asset.hostname) + '\nPemegang: ' + (asset.owner_username || 'Belum ditugaskan') + ' → ' + target.trim() + '\n\n' + (direct ? 'Konfirmasi serah terima dan langsung pindahkan tanggung jawab?' : 'Kirim permintaan untuk persetujuan ADH?'))) return;
  if (swapTag.trim() && !confirm('Tukar dua aset sekaligus dengan ' + swapTag.trim() + '? Kedua pemegang akan berubah setelah approval.')) return;
  var result = await api('/api/assets/switch-requests', {method:'POST', body:JSON.stringify({asset_id:id, asset_type:type, to_owner:target.trim(), reason:reason.trim(), swap_tag:swapTag.trim()})});
  if (!result || result.error) { alert((result && result.error) || 'Gagal mengirim switch'); return; }
  showToast(result.status === 'approved' ? 'Pemegang aset berhasil dipindahkan' : 'Permintaan menunggu persetujuan ADH');
  await loadBranchAssets();
  await openSwitchHistory();
}

async function openSwitchHistory() {
  document.getElementById('switchHistoryModal').style.display = 'flex';
  var box = document.getElementById('switchHistoryContent');
  box.textContent = 'Memuat...';
  var rows = await api('/api/assets/switch-requests?limit=100');
  if (!Array.isArray(rows)) { box.textContent = rows && rows.error || 'Tidak dapat memuat riwayat.'; return; }
  box.innerHTML = rows.length ? rows.map(function(r) {
    var actions = r.status === 'pending' && mayReviewSwitch() ? '<button class="btn btn-primary btn-sm" data-review="approve" data-id="'+r.id+'">Setujui</button> <button class="btn btn-danger btn-sm" data-review="reject" data-id="'+r.id+'">Tolak</button>' : '';
    var status = {pending:'Menunggu approval', approved:'Disetujui', rejected:'Ditolak'}[r.status] || r.status;
    if (r.swap_asset_id) status += ' · Tukar dua aset (' + r.swap_asset_id + ')';
    return '<div class="switch-entry"><strong>'+esc(r.asset_name)+'</strong> <span class="tag">'+esc(status)+'</span><p>'+esc(r.from_owner || 'Belum ditugaskan')+' → '+esc(r.to_owner)+' · '+esc(r.branch)+'</p><p>Alasan: '+esc(r.reason)+'</p><small>Diajukan '+esc(r.requested_by)+' · '+new Date(r.created_at).toLocaleString()+'</small>'+(r.reviewed_by ? '<p>Diproses '+esc(r.reviewed_by)+' · '+esc(r.review_note)+'</p>' : '')+'<div>'+actions+'</div></div>';
  }).join('') : '<p>Belum ada permintaan switch.</p>';
  box.querySelectorAll('[data-review]').forEach(function(button) { button.onclick = function() { reviewAssetSwitch(Number(button.dataset.id), button.dataset.review); }; });
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
