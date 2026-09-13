# Cara Pakai RemoteDesk

Setelah server jalan, buka browser dan akses dashboard.
Panduan ini menjelaskan semua yang bisa kamu lakukan.

---

## 1. Login

1. Buka browser (Chrome, Firefox, Edge, Safari, apapun)
2. Ketik alamat server: `http://IP-VPS-KAMU:8080`
3. Muncul halaman login
4. Isi:
   - **Username**: `admin` (atau sesuai yang kamu set)
   - **Password**: password yang kamu set waktu jalankan server
5. Klik **Sign In**

Kalau berhasil, kamu masuk ke Dashboard.

> **Lupa password?** Restart server dengan password baru:
> `rd-server -admin-pass password-baru`

---

## 2. Dashboard

Ini halaman utama. Di sini kamu bisa lihat ringkasan semua device.

### Kartu Statistik (Atas)

Ada 4 kotak di bagian atas:

| Kotak | Artinya |
|-------|---------|
| **Total Devices** | Berapa device yang sudah terdaftar |
| **Online** | Berapa device yang sedang nyala dan terhubung |
| **Offline** | Berapa device yang mati atau tidak terhubung |
| **Groups** | Berapa grup device yang kamu punya |

### Grafik (Tengah)

Ada 4 grafik:

1. **Device Status** — Lingkaran (donut) hijau dan merah.
   Hijau = online, merah = offline. Langsung keliatan persentasenya.

2. **OS Distribution** — Lingkaran warna-warni.
   Menunjukkan berapa device yang pakai Linux, Windows, macOS.

3. **Top 10 Memory Usage** — Bar horizontal biru.
   10 device yang RAM-nya paling banyak terpakai. Kalau ada yang hampir penuh,
   langsung keliatan di sini.

4. **Top 10 Disk Usage** — Bar horizontal kuning.
   10 device yang disk-nya paling banyak terpakai. Berguna untuk cegah disk penuh.

Semua grafik **update otomatis** setiap 30 detik.

### Tabel Recent Devices (Bawah)

Tabel 10 device terakhir yang terlihat. Kolom-kolomnya:

| Kolom | Artinya |
|-------|---------|
| Status | Bulatan hijau = online, abu-abu = offline |
| Hostname | Nama komputer |
| ID | ID unik device |
| OS | Sistem operasi dan arsitektur (contoh: linux/amd64) |
| IP | Alamat IP |
| CPU | Prosesor dan jumlah core |
| Memory | RAM terpakai / total |
| Disk | Disk terpakai / total |
| Group | Grup device ini |
| Last Seen | Kapan terakhir terlihat online |
| Actions | Tombol **Details** dan **Remote** |

---

## 3. Halaman Devices

Klik **Devices** di sidebar kiri untuk melihat semua device.

### Cari Device

Ketik di kotak **Search** untuk cari berdasarkan:
- Nama komputer (hostname)
- IP address
- Device ID

### Filter

- **Group**: Pilih grup tertentu dari dropdown
- **Status**: Pilih "Online" atau "Offline"

### Lihat Detail Device

Klik tombol **Details** di baris device manapun. Muncul popup yang menampilkan:
- Semua info hardware (OS, CPU, RAM, disk, IP)
- Status online/offline
- Tanggal pertama kali terdaftar

Di popup ini kamu juga bisa:

- **Edit Tags** — Tambah label, pisahkan pakai koma.
  Contoh: `production, web-server, jakarta`

- **Edit Group** — Ganti grup device.
  Contoh: `kantor-jakarta`, `datacenter-sg`, `staging`

- **Edit Notes** — Tulis catatan bebas.
  Contoh: `PIC: Budi, Lokasi: Lantai 3, Kontrak sampai 2027`

Klik **Save** untuk menyimpan.

### Hapus Device

Di popup detail, klik **Delete** lalu konfirmasi. Device dihapus dari database.

---

## 4. Halaman Asset Inventory

Klik **Asset Inventory** di sidebar. Ini tampilan khusus untuk pendataan aset hardware.

### Apa Bedanya dengan Halaman Devices?

Halaman Devices fokus ke monitoring (online/offline, resource usage).
Halaman Asset fokus ke **inventaris hardware** — semua spek lengkap dalam satu tabel.

### Kolom yang Ditampilkan

| Kolom | Contoh |
|-------|--------|
| Hostname | `server-prod-01` |
| ID | `server-prod-01-48372` |
| OS | `linux` |
| Arch | `amd64` |
| CPU | `Intel Xeon E5-2680 v4` |
| Cores | `4` |
| RAM Total | `8.0 GB` |
| Disk Total | `100.0 GB` |
| IP | `203.0.113.10` |
| Local IP | `192.168.1.10` |
| Group | `datacenter-sg` |
| Tags | `production`, `web` |
| Notes | `PIC: Budi` |
| Registered | `2026-08-20` |

### Export ke CSV (Excel)

Klik tombol **Export CSV**. Browser akan download file CSV yang bisa dibuka di:
- Microsoft Excel
- Google Sheets
- LibreOffice Calc

File bernama `assets_2026-08-24.csv` (tanggal otomatis).

Berguna untuk:
- Laporan inventaris ke manajemen
- Audit hardware
- Backup data aset

---

## 5. Halaman Remote Desktop

Klik **Remote Desktop** di sidebar.

### Cara Remote ke Komputer

1. Pilih device yang **online** (hijau) dari dropdown
2. Klik **Connect**
3. Tunggu status berubah jadi "Connected" (hijau)
4. Layar komputer remote muncul di area hitam
5. Klik pada layar untuk fokus, lalu bisa:
   - Gerakkan mouse
   - Klik
   - Ketik keyboard

### Putus Koneksi

Klik tombol **Disconnect** (merah).

### Cara Cepat

Di halaman Devices, klik tombol **Remote** di baris device yang online.
Otomatis pindah ke halaman Remote Desktop dan langsung connect.

### Yang Perlu Diketahui

- Remote desktop butuh agent yang support screen capture
- Semua data lewat server (bukan langsung P2P)
- Cocok untuk: troubleshooting, monitoring, cek kondisi
- Untuk kerja berat (edit video, gaming): pakai VNC/RDP langsung

---

## 6. Logout

Klik **Logout** di sidebar paling bawah. Token login dihapus dari browser.

---

## Tips

### Organisasi Device yang Rapi
- Buat grup berdasarkan lokasi: `kantor-jakarta`, `kantor-surabaya`
- Atau berdasarkan fungsi: `web-server`, `database`, `monitoring`
- Tambah tags untuk label fleksibel: `production`, `staging`, `critical`
- Tulis notes untuk info penting: PIC, lokasi fisik, nomor kontrak

### Monitoring Efektif
- Buka Dashboard, lihat grafik Memory dan Disk
- Device yang hampir penuh RAM/disk-nya langsung keliatan di Top 10
- Status online/offline update real-time (tidak perlu refresh manual)

### Keamanan
- Ganti password admin default secepatnya
- Pakai HTTPS (lihat panduan install)
- Jangan share API key sembarangan
- Batasi akses ke port 8080 dari IP tertentu saja (pakai firewall)
