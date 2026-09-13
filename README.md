# RemoteDesk

Sistem untuk **remote desktop** dan **catat data aset komputer** — mirip RustDesk tapi super ringan.

Bisa jalan di VPS murah (1 vCPU, 1 GB RAM) dan handle **1000+ device** sekaligus.

---

## Apa Itu RemoteDesk?

Bayangkan kamu punya 100 komputer di kantor / server di berbagai tempat.
Kamu pengen bisa:

1. **Lihat semua komputer** dari satu dashboard (online/offline, spek hardware, IP, dll)
2. **Remote desktop** — lihat layar dan kontrol komputer dari browser
3. **Catat aset** — data CPU, RAM, disk, OS otomatis terkumpul, bisa export Excel/CSV

RemoteDesk melakukan semua itu dengan **2 program kecil**:

| Program | Dipasang di mana | Fungsi |
|---------|-----------------|--------|
| `rd-server` | VPS / server pusat (1 buah) | Terima data dari semua komputer, tampilkan dashboard |
| `rd-agent` | Setiap komputer yang mau dikelola | Kirim info hardware, terima perintah remote |

---

## Hasil Build = File Apa? Ada di Mana?

Setelah di-build, kamu dapat **file executable** (program yang bisa langsung dijalankan).

File-file ini masuk ke folder `bin/` di dalam folder project:

```
remote-desktop/           ← folder project kamu
├── bin/                   ← ⭐ HASIL BUILD ADA DI SINI
│   ├── rd-server.exe     ← program server (Windows, ~22 MB)
│   └── rd-agent.exe      ← program agent (Windows, ~8 MB)
├── cmd/
├── internal/
├── web/
├── docs/
└── ...
```

**Di Linux/Mac**, file-nya tidak pakai `.exe`:
```
bin/
├── rd-server     ← program server (~20 MB)
└── rd-agent      ← program agent (~7 MB)
```

**File-file ini tidak perlu di-install.** Tinggal copy ke mana saja dan jalankan.

---

## Cara Build (Step by Step)

### Di Windows (PowerShell)

```powershell
# 1. Buka PowerShell, masuk ke folder project
cd D:\Project\remote-desktop

# 2. Download dependency (cuma perlu sekali)
go mod tidy

# 3. Build server — hasilnya: bin\rd-server.exe
go build -o bin\rd-server.exe ./cmd/server

# 4. Build agent — hasilnya: bin\rd-agent.exe
go build -o bin\rd-agent.exe ./cmd/agent

# 5. Cek hasilnya: buka folder bin
dir bin\
#    Kamu akan lihat:
#    rd-server.exe    22 MB
#    rd-agent.exe      8 MB

# 6. Mau buka folder bin di Explorer? Ketik:
explorer bin
#    Folder terbuka, kamu bisa lihat dan copy file-nya dari situ
```

### Di Linux / macOS (Terminal)

```bash
# 1. Masuk ke folder project
cd ~/remote-desktop

# 2. Download dependency (cuma perlu sekali)
go mod tidy

# 3. Build server — hasilnya: bin/rd-server
CGO_ENABLED=1 go build -o bin/rd-server ./cmd/server

# 4. Build agent — hasilnya: bin/rd-agent
go build -o bin/rd-agent ./cmd/agent

# 5. Cek hasilnya
ls -lh bin/
#    rd-server    20M
#    rd-agent      7M
```

### Build Agent untuk Linux dari Windows

Kamu build di Windows, tapi mau hasilnya jalan di Linux? Bisa:

```powershell
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="amd64"
go build -o bin\rd-agent-linux ./cmd/agent

# Hasilnya: bin\rd-agent-linux
# File ini kamu upload ke server Linux nanti
```

---

## Setelah Build, Terus Ngapain?

### Skenario: Server di VPS Linux, Agent di Komputer Windows

Ini skenario paling umum.

#### 1. Upload `rd-server` ke VPS

Kamu perlu copy file `rd-server` (versi Linux) dari komputer kamu ke VPS.

**Pakai SCP (dari PowerShell/terminal):**
```bash
scp bin\rd-agent-linux root@IP-VPS-KAMU:/usr/local/bin/rd-server
```

**Atau pakai WinSCP / FileZilla:**
- Connect ke VPS
- Navigate ke `/usr/local/bin/`
- Upload file `rd-server` (versi Linux) ke sana

#### 2. Jalankan server di VPS

SSH ke VPS, lalu:
```bash
chmod +x /usr/local/bin/rd-server
/usr/local/bin/rd-server -addr :8080 -admin-pass rahasia123 -api-key kunci-rahasia
```

Muncul: `[server] listening on :8080` → **server jalan!**

Buka browser: `http://IP-VPS-KAMU:8080` → muncul halaman login.

#### 3. Jalankan agent di komputer Windows

Buka folder `bin\` di project kamu (atau copy `rd-agent.exe` ke mana saja).

Buka Command Prompt / PowerShell **di folder yang ada rd-agent.exe**, lalu:

```
.\rd-agent.exe -server http://IP-VPS-KAMU:8080 -key kunci-rahasia
```

Muncul: `[agent] connected` dan `[agent] registered` → **komputer terdaftar!**

Buka dashboard di browser → komputer kamu sudah muncul di daftar. ✅

---

## Dokumentasi Lengkap

Panduan lebih detail ada di folder `docs/`:

| Dokumen | Isi |
|---------|-----|
| [Cara Install & Build](docs/01-installation.md) | Build, Docker, pasang di VPS, HTTPS |
| [Cara Pakai Dashboard](docs/02-user-guide.md) | Login, lihat device, grafik, edit aset, export CSV |
| [Cara Pasang Agent](docs/03-agent-deployment.md) | Pasang agent di Windows/Linux/macOS, biar jalan otomatis |
| [API Reference](docs/04-api-reference.md) | Daftar API untuk script/otomasi |
| [Arsitektur & Performa](docs/05-architecture.md) | Kenapa bisa handle 1000 device di VPS kecil |
| [Troubleshooting](docs/06-troubleshooting.md) | Kalau ada masalah, baca ini |

---

## Fitur

- ✅ Dashboard real-time (online/offline, grafik, statistik)
- ✅ Grafik Chart.js (status device, distribusi OS, pemakaian RAM & disk)
- ✅ Inventaris aset otomatis (CPU, RAM, disk, OS, IP)
- ✅ Export data ke CSV / Excel
- ✅ Remote desktop dari browser
- ✅ Grouping & tagging device
- ✅ REST API lengkap
- ✅ Login admin dengan password
- ✅ Super ringan (1 VPS murah cukup untuk 1000 device)
- ✅ Single binary — tidak perlu install, tidak perlu database server

---

## Tech Stack

| Komponen | Teknologi |
|----------|-----------|
| Server & Agent | Go |
| Database | SQLite (embedded, tidak perlu install terpisah) |
| Dashboard | HTML + CSS + JavaScript (embedded di server) |
| Grafik | Chart.js |
| Komunikasi | WebSocket |

## License

MIT
