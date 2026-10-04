# Rotasi API key bertahap (v0.2.62)

- Menu **Ubah Endpoint / API Key**: siapkan key tanpa broadcast, pilih satu perangkat pilot, lalu migrasikan perangkat lain satu per satu.
- Key menerima simbol, spasi, dan Unicode; panjang 16-512 byte, tanpa karakter kontrol. Gunakan key acak yang panjang. Agent v0.2.62 meng-encode parameter URL; agent lebih lama ditolak untuk migrasi, sehingga harus di-update dahulu.
- Key lama tetap menerima koneksi. Tidak ada migrasi otomatis ketika perangkat kembali online. Tombol migrasi hanya mengirim ke perangkat terpilih yang sedang online.
- **Terima key lama tambahan** menerima key legacy pendek khusus pemulihan (tidak kosong, maksimal 512 byte, tanpa karakter kontrol). Key aktif tidak berubah. Gunakan tombol ini untuk key 11 byte yang sudah dipakai PC, bukan **Siapkan key baru**. Tidak perlu memilih perangkat atau mengisi endpoint. Setelah pulih, migrasikan ke key baru minimal 16 byte.
- Badge **Key terbaru terverifikasi**, **Key lama**, dan **Key belum terverifikasi** berasal dari koneksi terautentikasi terakhir, bukan dari pengiriman perintah. Status offline tidak membuktikan isi konfigurasi terbaru perangkat.
- Key aktif, key lama, dan verifikasi perangkat tersimpan di SQLite pada volume data. RD_API_KEY hanya bootstrap ketika belum ada state rotasi. Setelah itu mengubah YAML tidak mengubah key aktif atau menghidupkan kembali key yang sudah dicabut.
- **Cabut semua key lama** memerlukan seluruh device terdaftar sudah terverifikasi dengan key terbaru. Device offline yang belum migrasi tetap memblokir pencabutan. Perangkat yang benar-benar dipensiunkan perlu dihapus secara sadar dari inventaris sebelum pencabutan.
- Agent menyimpan konfigurasi pengganti melalui file sementara. Jika koneksi dengan key baru gagal, agent memulihkan key sebelumnya dan mencoba kembali. Jangan mencabut key lama selama pilot belum stabil.
- Service Windows versi baru menghentikan worker miliknya sebelum berhenti. Pembaruan dari service lama mungkin masih memerlukan penghentian worker tertinggal sekali secara manual.

## Upgrade dari implementasi lama

State key implementasi lama hanya berada di memori proses; rilis baru tidak bisa mengambilnya setelah proses tersebut dihentikan. Sebelum deployment pertama, pastikan key aktif produksi tersedia secara aman dan samakan RD_API_KEY dengan key aktif itu untuk bootstrap. Setelah deployment, gunakan **Terima key lama tambahan** untuk mendaftarkan kembali key awal yang masih dipakai perangkat offline. Jangan menyalin key ke tiket, chat, screenshot, atau log. Jangan mengasumsikan key runtime lama otomatis terbawa oleh update container.

Lindungi akses volume database dan backup: state rotasi memuat credential untuk migrasi agent. UI status hanya mengembalikan jumlah key lama, bukan nilainya.

## Verifikasi

Tes server mencakup tidak adanya broadcast, migrasi satu device, penolakan device offline/agent lama, dua rotasi, restart dengan YAML berbeda, pencabutan, dan WebSocket key lama tanpa auto-migrasi. Tes agent mencakup encoding simbol, kegagalan simpan, serta rollback setelah handshake ditolak. Tes Windows menghentikan proses uji milik supervisor; bukan service produksi pengguna.
