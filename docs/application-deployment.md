# Deploy aplikasi dan transfer file — 0.2.56

## Deploy aplikasi

1. Perbarui server dan agent ke 0.2.56. Hanya admin dan IT Support.
2. Devices > Deploy aplikasi. Pilih MSI/EXE (maksimum 100 MiB).
3. MSI menggunakan /qn /norestart; argumen tambahan belum didukung. EXE memerlukan array JSON argumen silent/no-restart yang sesuai dokumentasi vendor. Tidak ada flag universal. Jangan memasukkan password atau token.
4. Filter cabang dan centang 1–50 perangkat Windows online. Daftar chooser memuat maksimal 10.000 perangkat, tidak mengikuti pagination tabel Devices.
5. Periksa konfirmasi nama installer dan semua target. Status tersimpan di SQLite dan diperbarui tiap lima detik saat modal terbuka.

Installer dijalankan oleh akun proses agent; instalasi melalui service LocalSystem memakai SYSTEM. Tidak membuka sesi remote desktop. File diunduh dari server menggunakan token job acak berumur maksimal 30 menit, tanpa redirect; SHA-256 diverifikasi sebelum menjalankan installer. Checksum bukan bukti bahwa installer aman: gunakan sumber vendor tepercaya dan uji satu perangkat terlebih dahulu.

Satu instalasi aktif per proses agent; permintaan lain ditolak, bukan diulang otomatis. Tidak mendukung antrean perangkat offline. Timeout total 25 menit; unduhan maksimal 5 menit. Timeout/kehilangan hasil menghasilkan status unknown: installer turunan mungkin masih berjalan. Jangan mencoba ulang sebelum memeriksa target. Job yang belum dikirim saat server restart tidak dikirim ulang otomatis.

Exit 0 = installer melaporkan berhasil, bukan verifikasi aplikasi telah terpasang. Exit 3010 = restart diperlukan; 1641 = installer melaporkan restart dimulai. EXE dapat mengabaikan argumen no-restart. Tidak ada reboot tambahan yang dikirim sistem.

Paket tersimpan dalam folder deploy-packages di sebelah database server. Retensi otomatis belum diterapkan; perhitungkan kapasitas disk dan hapus paket lama melalui pemeliharaan setelah semua job terkait final/kedaluwarsa. Token job final tidak dapat mengunduh paket lagi.

## Transfer file

Remote Web > Kirim File membuka dua panel. Panel lokal memakai pemilih file browser, bukan akses bebas seluruh filesystem lokal. Panel remote menampilkan maksimal 500 entri folder; folder dapat dibuka, naik ke induk, atau ditentukan melalui path absolut lokal. Folder tujuan harus sudah ada. UNC/network share tidak didukung.

Pilih file di kiri, pilih folder kanan, lalu Kirim ke folder kanan. Maksimum 100 MiB. File disiapkan di subfolder staging tujuan, diverifikasi checksum, lalu dipublikasikan tanpa menimpa file yang sudah ada. Publikasi membutuhkan filesystem yang mendukung hard link (misalnya NTFS); kegagalan tidak menimpa file lama. Transfer masih satu arah, lokal ke remote.

## Batas verifikasi

Tes Go memeriksa otorisasi, dispatch dua target, akses paket, pengikatan hasil ke device, checksum/transfer dan larangan overwrite. Tes Node memeriksa alur upload dan UI. Installer sungguhan belum dijalankan pada fleet produksi.
