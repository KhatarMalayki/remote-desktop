# Arsitektur & Kenapa Bisa Ringan

Dokumen ini menjelaskan kenapa RemoteDesk bisa handle 1000 device di VPS kecil.
Kalau kamu cuma mau pakai, **tidak perlu baca ini** — ini untuk yang penasaran teknis.

---

## Gambaran Besar

```
                    ╔══════════════════════════════════╗
                    ║        VPS Kamu (1 vCPU, 1 GB)   ║
                    ║                                  ║
                    ║   ┌────────────────────────────┐ ║
                    ║   │       rd-server             │ ║
                    ║   │                            │ ║
                    ║   │  • Terima koneksi agent    │ ║
                    ║   │  • Simpan data ke SQLite   │ ║
                    ║   │  • Tampilkan dashboard web │ ║
                    ║   │  • Relay remote desktop    │ ║
                    ║   └─────────────┬──────────────┘ ║
                    ║                 │                 ║
                    ║   ┌─────────────┴──────────────┐ ║
                    ║   │    SQLite (file database)   │ ║
                    ║   │    devices.db (~10 MB)      │ ║
                    ║   └────────────────────────────┘ ║
                    ╚═══════════════╤══════════════════╝
                                   │
              ┌────────────────────┼───────────────────┐
              │                    │                    │
         ┌────┴────┐         ┌────┴────┐         ┌────┴────┐
         │ Agent 1 │         │ Agent 2 │         │ Browser │
         │ (Linux) │         │(Windows)│         │ (Admin) │
         └─────────┘         └─────────┘         └─────────┘
```

---

## Kenapa Pakai Go?

Go dipilih karena:
- **Ringan**: Program Go tidak butuh runtime besar (beda dengan Java/Python)
- **Goroutine**: Bisa handle ribuan koneksi pakai memori sangat kecil
- **Single binary**: Hasil compile = 1 file, tidak perlu install apapun
- **Cross-compile**: Build untuk Linux/Windows/Mac dari satu komputer

---

## Kenapa Pakai SQLite (Bukan MySQL/PostgreSQL)?

- **Tidak perlu install** — SQLite itu cuma 1 file (`.db`)
- **Tidak perlu server database** — hemat RAM
- **WAL mode** — bisa baca bersamaan, tulis bergantian
- Untuk 1000-5000 device, SQLite **lebih dari cukup**
- Kalau butuh > 10.000 device, baru pertimbangkan PostgreSQL

---

## Hitungan Memori: Kok Bisa 1000 Device di 1 GB?

### Setiap koneksi agent pakai berapa memori?

```
1 agent = 1 koneksi WebSocket = 2 goroutine (baca + tulis)

1 goroutine = ~4 KB memori
2 goroutine = ~8 KB

Buffer WebSocket: 4 KB (baca) + 4 KB (tulis) = 8 KB

Total per agent: ~16 KB
```

### Jadi untuk 1000 device:

```
Koneksi agent:    1000 × 16 KB  = ~16 MB
Program server:                 = ~20 MB
SQLite database:                = ~10 MB
Dashboard web:                  = ~2 MB (embedded)
Sistem operasi:                 = ~100 MB
────────────────────────────────────────
Total:                          ≈ 150 MB dari 1024 MB (1 GB)

Sisa: ~870 MB untuk yang lain! 👍
```

### CPU Usage

```
Heartbeat: 1000 device × 1 request / 30 detik = ~33 writes/detik ke SQLite
Bandwidth: 1000 device × 100 bytes / 30 detik = ~3.3 KB/detik

Untuk 1 vCPU, ini sangat ringan (CPU usage < 5% di idle).
```

---

## Alur Kerja (Flow)

### Agent Connect → Register

```
Agent start
  → Koneksi WebSocket ke server
  → Kirim info hardware (hostname, OS, CPU, RAM, disk, IP)
  → Server simpan ke SQLite
  → Server broadcast "device X online" ke semua dashboard yang buka
  → Dashboard update otomatis (tanpa refresh)
```

### Heartbeat (Setiap 30 Detik)

```
Agent kirim: {"cpu_usage": 25, "memory_used": 4GB, "disk_used": 50GB}
  → Server update SQLite (UPDATE ... WHERE id = ?)
  → Dashboard refresh data
```

### Remote Desktop

```
Admin klik "Connect" di dashboard
  → Dashboard bikin session relay di server
  → Minta agent connect ke session yang sama
  → Agent kirim screenshot (JPEG) ke server
  → Server forward ke dashboard
  → Admin lihat layar
  → Admin kirim mouse/keyboard event
  → Server forward ke agent
  → Agent execute mouse/keyboard di komputer
```

Semua data relay lewat server (bukan langsung P2P).

---

## Teknologi yang Dipakai

| Komponen | Teknologi | Kenapa |
|----------|-----------|--------|
| Server | Go `net/http` | Bawaan Go, tidak perlu framework berat |
| WebSocket | gorilla/websocket | Library paling mature untuk Go |
| Database | SQLite + go-sqlite3 | Embedded, zero config |
| Frontend | Vanilla HTML/CSS/JS | Tanpa React/Vue/Angular = tidak perlu build step |
| Charts | Chart.js (CDN) | Library chart ringan, cantik, gratis |
| Auth | SHA256 HMAC | Sederhana, cukup untuk internal |
| Binary embed | Go `embed` | Web files dikemas dalam 1 binary |

---

## Limitasi (Yang Tidak Bisa)

| Limitasi | Penjelasan | Solusi |
|----------|------------|--------|
| Remote desktop berat | Semua traffic lewat server | Untuk kerja berat, pakai RDP/VNC langsung |
| > 5000 device | SQLite mulai lambat untuk write-heavy | Migrasi ke PostgreSQL |
| Multi-user auth | Cuma 1 admin user | Bisa dikembangkan nanti |
| HTTPS built-in | Server cuma HTTP | Pakai Nginx/Caddy di depan |
| Audit trail | Log sederhana | Bisa tambah logging lebih detail |
