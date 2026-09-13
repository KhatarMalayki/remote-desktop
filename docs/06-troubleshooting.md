# Troubleshooting (Kalau Ada Masalah)

Cari masalah kamu di daftar di bawah. Setiap masalah ada solusinya.

---

## Masalah Server

### ❌ Server tidak mau start, error "address already in use"

**Artinya:** Port 8080 sudah dipakai program lain.

**Solusi:**
```bash
# Cek siapa yang pakai port 8080
# Linux:
sudo ss -tlnp | grep 8080

# Windows:
netstat -an | findstr 8080

# Opsi 1: Matikan program yang pakai port itu
# Opsi 2: Ganti port server
./rd-server -addr :9090    # pakai port 9090 misalnya
```

### ❌ Buka browser tapi halaman tidak muncul

**Cek satu-satu:**

1. Server jalan? Cek di terminal, harusnya ada tulisan `[server] listening on :8080`

2. Alamat benar? Pastikan `http://IP-VPS:8080` (pakai http, bukan https)

3. Port terbuka? Cek firewall VPS:
   ```bash
   # Ubuntu/Debian:
   sudo ufw allow 8080

   # CentOS/RHEL:
   sudo firewall-cmd --add-port=8080/tcp --permanent
   sudo firewall-cmd --reload
   ```

4. Coba dari VPS sendiri:
   ```bash
   curl http://localhost:8080
   # Harus muncul HTML
   ```
   Kalau dari VPS bisa tapi dari luar tidak bisa = masalah firewall.

### ❌ Login gagal "invalid credentials"

**Penyebab:** Password salah.

**Solusi:**
- Pastikan password yang kamu ketik sama persis dengan yang di-set waktu start server (`-admin-pass`)
- Password itu case-sensitive (huruf besar/kecil beda)
- Kalau lupa: restart server dengan password baru

### ❌ API return "unauthorized" (401)

**Penyebab:** Token expired (berlaku 24 jam) atau salah.

**Solusi:**
- Login lagi di dashboard (otomatis dapat token baru)
- Atau untuk script, pakai API Key langsung (tidak expire)

---

## Masalah Agent

### ❌ Agent tidak mau connect, "connection failed"

**Cek satu-satu:**

1. **Server jalan?** Buka `http://IP-VPS:8080` di browser. Bisa? Lanjut.

2. **URL benar?** Harus lengkap termasuk `http://`:
   ```
   ✅ Benar: -server http://103.15.20.5:8080
   ❌ Salah: -server 103.15.20.5:8080
   ❌ Salah: -server https://103.15.20.5:8080  (kalau servernya HTTP)
   ```

3. **API Key cocok?** API key di agent harus sama persis dengan yang di server.

4. **Bisa ping?** Dari komputer agent, coba:
   ```bash
   ping IP-VPS
   curl http://IP-VPS:8080
   ```
   Kalau tidak bisa ping/curl = masalah jaringan atau firewall.

### ❌ Agent connect tapi device tidak muncul di dashboard

**Cek:**
- Di terminal agent, ada tulisan `[agent] registered`? Kalau ada, harusnya muncul.
- Refresh halaman dashboard (F5)
- Cek di halaman Devices, mungkin perlu scroll atau ganti page

### ❌ Agent sering putus dan reconnect

**Penyebab umum:**
- Internet tidak stabil
- Firewall memutus koneksi idle
- Server overload

**Solusi:**
- Kalau pakai Nginx/reverse proxy, pastikan:
  ```nginx
  proxy_read_timeout 86400;  # 24 jam
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection "upgrade";
  ```
- Naikkan heartbeat: `-heartbeat 60` (lebih jarang kirim data)

### ❌ Info hardware salah/kosong

**Windows:**
- Jalankan agent sebagai **Administrator** (klik kanan → Run as administrator)
- Pastikan WMI service jalan: `services.msc` → cari "Windows Management Instrumentation" → Start

**Linux:**
- Pastikan file `/proc/cpuinfo` dan `/proc/meminfo` bisa dibaca
- Pastikan perintah `df` tersedia

---

## Masalah Remote Desktop

### ❌ Klik Connect tapi layar tidak muncul

- Remote desktop butuh agent yang support screen capture
- Agent dasar cuma kirim info hardware, fitur screen capture masih dalam development
- Pastikan device target online (hijau di dashboard)

### ❌ Remote desktop lambat/patah-patah

- Semua data lewat server (bukan langsung), jadi:
  - Bandwidth server jadi faktor utama
  - Jarak server ke agent juga berpengaruh
- Untuk kerja berat: pakai RDP (Windows) atau VNC (Linux) langsung

---

## Masalah Docker

### ❌ Container tidak start

```bash
# Lihat log error
docker logs remotedesk

# Biasanya: port conflict, permission error, atau env variable salah
```

### ❌ Data hilang setelah restart container

Pastikan pakai volume!
```bash
docker run -v remotedesk-data:/data ...
#           ^^^^^^^^^^^^^^^^^^^^^^^^ ini penting!
```

Tanpa `-v`, data hilang setiap container restart.

---

## Masalah Build

### ❌ Error "gcc not found" waktu build server

Server butuh GCC karena pakai SQLite (CGO):

```bash
# Ubuntu/Debian:
sudo apt install gcc

# macOS:
xcode-select --install

# Windows:
# Install TDM-GCC: https://jmeubank.github.io/tdm-gcc/
```

### ❌ Error "go: command not found"

Go belum di-install atau belum masuk PATH:
- Download Go: https://go.dev/dl/
- Setelah install, restart terminal

---

## Masih Bingung?

1. Cek log server: biasanya ada pesan error yang jelas
   ```bash
   # Kalau pakai systemd:
   journalctl -u remotedesk -f

   # Kalau pakai Docker:
   docker logs -f remotedesk
   ```

2. Cek log agent: biasanya tertulis kenapa gagal connect

3. Buka browser Developer Tools (F12) → tab Console → lihat error JavaScript

4. Pastikan versi server dan agent sama (build dari source yang sama)
