# Cara Install RemoteDesk

Panduan ini ditulis selangkah demi selangkah. Ikuti dari atas ke bawah.

---

## Pertama: Pahami Dulu

RemoteDesk itu **2 program kecil**:

1. **`rd-server`** = Program server. Dipasang **1 kali saja** di VPS/server pusat kamu.
   Ini yang menampilkan dashboard web dan menerima data dari semua komputer.

2. **`rd-agent`** = Program agent. Dipasang di **setiap komputer** yang mau kamu kelola.
   Ini yang mengirim info hardware (CPU, RAM, disk, dll) ke server.

Kedua program ini **tidak perlu di-install**. Tidak ada installer, tidak ada setup wizard.
Cukup jalankan file-nya, selesai.

---

## Cara Dapat File Program-nya

Ada 3 cara:

### Cara A: Build Sendiri dari Source Code (Perlu Go)

Ini artinya kamu compile sendiri kode program jadi file `.exe` (Windows) atau binary (Linux/Mac).

**Yang kamu butuhkan:**
- Go 1.22 atau lebih baru (download di https://go.dev/dl/)
- GCC compiler (untuk server saja, karena pakai SQLite)
  - Windows: Install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) atau [MSYS2](https://www.msys2.org/)
  - Linux: `sudo apt install gcc` (biasanya sudah ada)
  - macOS: `xcode-select --install`

**Langkah-langkah:**

```bash
# 1. Masuk ke folder project
cd remote-desktop

# 2. Download dependency
go mod tidy

# 3. Build server
#    Hasilnya: file "rd-server" (Linux/Mac) atau "rd-server.exe" (Windows)
#    di dalam folder bin/
go build -o bin/rd-server ./cmd/server      # Linux/Mac
go build -o bin/rd-server.exe ./cmd/server   # Windows

# 4. Build agent
#    Hasilnya: file "rd-agent" (Linux/Mac) atau "rd-agent.exe" (Windows)
go build -o bin/rd-agent ./cmd/agent         # Linux/Mac
go build -o bin/rd-agent.exe ./cmd/agent     # Windows
```

**Setelah selesai**, cek folder `bin/`:
```
bin/
├── rd-server.exe    ← ini program server (~22 MB)
└── rd-agent.exe     ← ini program agent (~8 MB)
```

**Mau build agent untuk Linux dari Windows?** Bisa! Namanya cross-compile:
```powershell
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="amd64"
go build -o bin/rd-agent-linux ./cmd/agent
# Hasilnya: file rd-agent-linux (untuk Linux 64-bit)
```

### Cara B: Pakai Docker (Untuk Server Saja)

Kalau VPS kamu sudah ada Docker, ini paling gampang:

```bash
# 1. Build image Docker
docker build -t remotedesk .

# 2. Jalankan
docker run -d \
  --name remotedesk \
  -p 8080:8080 \
  -v remotedesk-data:/data \
  -e RD_ADMIN_PASS=ganti-password-ini \
  -e RD_API_KEY=ganti-kunci-ini \
  --restart unless-stopped \
  remotedesk
```

Selesai! Buka browser: `http://IP-VPS:8080`

### Cara C: Pakai Docker Compose (Paling Simple)

```bash
# 1. Edit file docker-compose.yml, ganti password dan API key
# 2. Jalankan:
docker compose up -d

# Mau lihat log:
docker compose logs -f

# Mau stop:
docker compose down
```

---

## Pasang Server di VPS Linux (Tanpa Docker)

Ini cara paling umum: kamu punya VPS Linux (Ubuntu, Debian, dll) dan mau jalankan server.

### Langkah 1: Upload file rd-server ke VPS

Dari komputer kamu, upload file `rd-server` (yang sudah di-build untuk Linux) ke VPS:

```bash
# Dari komputer lokal:
scp bin/rd-server-linux root@IP-VPS-KAMU:/usr/local/bin/rd-server
```

Atau pakai FileZilla / WinSCP kalau lebih nyaman.

### Langkah 2: Beri izin execute

```bash
# Di VPS:
chmod +x /usr/local/bin/rd-server
```

### Langkah 3: Jalankan server (tes dulu)

```bash
/usr/local/bin/rd-server \
  -addr :8080 \
  -admin-user admin \
  -admin-pass password-kamu \
  -api-key kunci-rahasia-kamu
```

Kalau muncul tulisan `[server] listening on :8080` berarti **berhasil**!

Buka browser: `http://IP-VPS-KAMU:8080`
- Login dengan username `admin` dan password yang kamu set tadi

> **PENTING:** Catat **API Key** yang muncul di log! Agent butuh ini untuk connect.
> Kalau kamu set sendiri via `-api-key`, ya pakai yang itu.

### Langkah 4: Bikin Jalan Otomatis (Systemd)

Supaya server jalan otomatis waktu VPS restart:

```bash
# 1. Buat folder data
sudo mkdir -p /var/lib/remotedesk

# 2. Buat user khusus (opsional, untuk keamanan)
sudo useradd -r -s /usr/sbin/nologin remotedesk
sudo chown remotedesk:remotedesk /var/lib/remotedesk

# 3. Buat file service
sudo nano /etc/systemd/system/remotedesk.service
```

Isi file-nya (copy-paste, ganti password dan api-key):

```ini
[Unit]
Description=RemoteDesk Server
After=network.target

[Service]
Type=simple
User=remotedesk
WorkingDirectory=/var/lib/remotedesk
ExecStart=/usr/local/bin/rd-server -addr :8080 -db /var/lib/remotedesk/devices.db -admin-pass PASSWORD-KAMU -api-key KUNCI-KAMU
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Simpan (Ctrl+O, Enter, Ctrl+X), lalu:

```bash
# 4. Aktifkan dan jalankan
sudo systemctl daemon-reload
sudo systemctl enable remotedesk
sudo systemctl start remotedesk

# 5. Cek apakah jalan
sudo systemctl status remotedesk
# Harus muncul "active (running)" warna hijau
```

Sekarang server akan **jalan otomatis** setiap VPS nyala!

---

## Pasang HTTPS (Biar Aman)

Kalau mau akses dashboard pakai `https://` (disarankan untuk production):

### Pakai Caddy (Paling Gampang, Auto SSL)

```bash
# 1. Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy

# 2. Buat config
sudo nano /etc/caddy/Caddyfile
```

Isi:
```
domain-kamu.com {
    reverse_proxy localhost:8080
}
```

```bash
# 3. Jalankan
sudo systemctl restart caddy
```

Selesai! Caddy otomatis dapat sertifikat SSL dari Let's Encrypt.
Sekarang akses `https://domain-kamu.com`

---

## Checklist Setelah Install

- [ ] Server jalan dan bisa diakses dari browser
- [ ] Bisa login ke dashboard dengan username/password
- [ ] API key sudah dicatat (untuk dipasang di agent)
- [ ] Port 8080 terbuka di firewall VPS
- [ ] (Opsional) HTTPS sudah aktif
- [ ] (Opsional) Systemd service sudah aktif

Langkah selanjutnya: [Cara Pasang Agent](03-agent-deployment.md)
