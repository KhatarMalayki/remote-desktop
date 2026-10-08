# RemoteDesk: paket perbaikan untuk uji coba

## Perubahan dan bukti

- Idle browser: aktivitas antar-tab dibaca melalui timestamp bersama; timeout tetap 30 menit. Tes regresi sebelumnya gagal ketika tab dashboard idle sementara tab remote aktif; kini lulus. Autentikasi server dan logout karena HTTP 401 tidak dilewati.
- VPN: watchdog sesi lama dapat menganggap lease sesi baru tidak valid. Watchdog kini keluar jika lease baru valid; lock file Windows menyerialisasi startup dengan pemeriksaan/penghentian watchdog. Tes native hanya menyentuh lock pada direktori sementara, bukan service VPN nyata.
- VPN: keepalive dikirim sebelum pemeriksaan handshake hub; handshake yang tidak terverifikasi tidak ditampilkan sebagai connected. Setelah timeout 60 detik sejak konfigurasi koneksi, peer dicabut dan alasan ditampilkan. Tombol Disconnect tersedia saat preparing/connecting.
- Metadata: perangkat agent memiliki serial_number dan product_id terpisah dari tags/note. Nomor produk berarti label produk laptop, bukan product key Windows. Migrasi, penyimpanan, pencarian, pembatasan peran, pembaruan parsial, dan pembukaan ulang database diuji. Heartbeat agent tidak menimpa nilai manual.
- Performa: resize JPEG memakai ApproxBiLinear menggantikan CatmullRom; polling hanya untuk halaman terlihat, tanpa polling dashboard/aset pada sesi remote aktif. Status connected tidak ditulis ulang pada setiap frame. Statistik menampilkan FPS dan Mbps data frame yang diterima.

## Pengukuran lokal

BenchmarkRemoteFrame4K: gambar sintetis RGBA 3840x2160, output 1600 piksel, kualitas JPEG 52. Windows amd64, Intel Core i7-1255U; tiga pengulangan dengan tiga iterasi masing-masing.

- Sebelum: 434, 383, 380 ms/op.
- Sesudah: 60, 58, 65 ms/op.
- Ini waktu resize dan encoding lokal, bukan latensi end-to-end, throughput VPN, atau perbandingan langsung RustDesk. Resize lebih ringan dapat mengubah ketajaman hasil downscale.

## Verifikasi dan batas

- node --test web/*.test.cjs: 37 tes lulus.
- go test ./internal/agent ./internal/server ./internal/vpn -count=1: lulus.
- Chromium: form SN/Product ID disimpan dan dibuka ulang setelah reload; API/autentikasi dimock. Persistensi SQLite diuji terpisah melalui tes Go.
- Chromium: dua tab mempertahankan sesi saat ada aktivitas, lalu keduanya kedaluwarsa saat idle; jam/autentikasi dimock.
- Koneksi WireGuard nyata, port UDP/firewall NAS, kualitas remote pada desktop pengguna, dan deployment belum diverifikasi. Akses lease VPN lokal ditolak ACL; tidak ada perubahan ACL atau restart service.
- Penerapan memerlukan server baru untuk UI/database/protokol hub dan agent Windows baru untuk resize/watchdog. Versi server dan agent: 0.2.67. Publikasi image dan kesehatan deployment harus diverifikasi terpisah setelah push.

## Uji setelah penerapan yang diotorisasi

1. Cadangkan database; jalankan server baru dan pastikan SN/Product ID bertahan setelah refresh.
2. Perbarui agent target. Connect VPN, pastikan handshake terverifikasi, lalu Disconnect dan segera Connect ulang. Jika gagal, kumpulkan alasan UI dan log hub/agent pada waktu yang sama.
3. Remote layar bergerak; catat profil kualitas, resolusi, FPS, Mbps, CPU agent, dan kondisi jaringan. Bandingkan pada pekerjaan yang sama.
4. Biarkan dashboard di tab lain selama lebih dari 30 menit sambil aktif di Remote Web; kemudian berhenti beraktivitas di semua tab selama 30 menit dan pastikan logout tetap terjadi.
