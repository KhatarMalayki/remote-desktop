# API Reference

Ini daftar semua API yang bisa kamu panggil dari program lain (script, Postman, curl, dll).
Berguna kalau kamu mau bikin integrasi atau otomasi.

> Kalau kamu cuma pakai dashboard web, **tidak perlu baca ini**.
> Ini untuk developer/sysadmin yang mau akses data via script.

---

## Cara Autentikasi

Setiap request ke API harus ada "tanda pengenal". Ada 2 cara:

### Cara 1: Pakai Token dari Login

Login dulu, dapat token, pakai token itu di setiap request:

```bash
# 1. Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password-kamu"}'

# Dapat response:
# {"token":"abcdef123456.signature","role":"admin"}

# 2. Pakai token-nya
curl http://localhost:8080/api/devices \
  -H "Authorization: Bearer abcdef123456.signature"
```

Token berlaku **24 jam**. Setelah itu harus login lagi.

### Cara 2: Pakai API Key Langsung

Lebih simple, tidak perlu login dulu:

```bash
curl http://localhost:8080/api/devices \
  -H "Authorization: Bearer KUNCI-API-KAMU"
```

API Key tidak expire. Cocok untuk script/cron job.

---

## Daftar API

### Login

```
POST /api/auth/login
```

Kirim username dan password, dapat token.

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

Response:
```json
{"token": "xxx.yyy", "role": "admin"}
```

---

### List Semua Device

```
GET /api/devices
```

```bash
# Semua device
curl -H "Authorization: Bearer TOKEN" http://localhost:8080/api/devices

# Cari device berdasarkan nama
curl -H "Authorization: Bearer TOKEN" "http://localhost:8080/api/devices?search=server1"

# Filter berdasarkan grup
curl -H "Authorization: Bearer TOKEN" "http://localhost:8080/api/devices?group=jakarta"

# Paginasi (halaman ke-2, 10 per halaman)
curl -H "Authorization: Bearer TOKEN" "http://localhost:8080/api/devices?limit=10&offset=10"
```

Response:
```json
{
  "devices": [
    {
      "id": "server1-12345",
      "hostname": "server1",
      "os": "linux",
      "arch": "amd64",
      "ip": "203.0.113.10",
      "local_ip": "192.168.1.10",
      "cpu_model": "Intel Xeon E5-2680",
      "cpu_cores": 4,
      "memory_total": 8589934592,
      "memory_used": 4294967296,
      "disk_total": 107374182400,
      "disk_used": 53687091200,
      "version": "",
      "status": "active",
      "tags": "production,web",
      "group": "datacenter-sg",
      "note": "Web server utama",
      "last_seen": "2026-08-24T10:30:00Z",
      "registered_at": "2026-08-20T08:00:00Z",
      "online": true
    }
  ],
  "total": 150,
  "limit": 50,
  "offset": 0
}
```

> **Catatan:** `memory_total`, `memory_used`, `disk_total`, `disk_used` dalam **bytes**.
> Contoh: `8589934592` bytes = 8 GB.

---

### Detail Satu Device

```
GET /api/devices/{id}
```

```bash
curl -H "Authorization: Bearer TOKEN" http://localhost:8080/api/devices/server1-12345
```

Response: sama seperti object device di atas.

---

### Update Info Device (Tags, Group, Note)

```
PUT /api/devices/{id}
```

```bash
curl -X PUT -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tags":"production,critical","group":"dc-sg","note":"Server utama"}' \
  http://localhost:8080/api/devices/server1-12345
```

Response: `{"status": "ok"}`

---

### Hapus Device

```
DELETE /api/devices/{id}
```

```bash
curl -X DELETE -H "Authorization: Bearer TOKEN" \
  http://localhost:8080/api/devices/server1-12345
```

Response: `{"status": "deleted"}`

---

### Statistik Dashboard

```
GET /api/stats
```

```bash
curl -H "Authorization: Bearer TOKEN" http://localhost:8080/api/stats
```

Response:
```json
{
  "total_devices": 150,
  "online_devices": 142,
  "offline_devices": 8,
  "os_distribution": {"linux": 120, "windows": 25, "darwin": 5}
}
```

---

### List Semua Group

```
GET /api/groups
```

```bash
curl -H "Authorization: Bearer TOKEN" http://localhost:8080/api/groups
```

Response:
```json
["datacenter-sg", "default", "office-jakarta"]
```

---

### Activity Log Device

```
GET /api/logs/{device_id}
```

```bash
curl -H "Authorization: Bearer TOKEN" "http://localhost:8080/api/logs/server1-12345?limit=10"
```

Response:
```json
[
  {"id": 1, "device_id": "server1-12345", "action": "register", "detail": "server1 linux", "created_at": "2026-08-20T08:00:00Z"},
  {"id": 2, "device_id": "server1-12345", "action": "update_meta", "detail": "tags=prod group=dc-sg", "created_at": "2026-08-21T09:15:00Z"}
]
```

---

## Contoh Script: Monitoring Otomatis

Script bash yang cek device offline dan kirim notifikasi:

```bash
#!/bin/bash
API_KEY="kunci-api-kamu"
SERVER="http://localhost:8080"

# Ambil statistik
stats=$(curl -s -H "Authorization: Bearer $API_KEY" $SERVER/api/stats)
offline=$(echo $stats | python3 -c "import sys,json; print(json.load(sys.stdin)['offline_devices'])")

if [ "$offline" -gt 0 ]; then
  echo "PERINGATAN: Ada $offline device offline!"
  # Bisa tambah: kirim email, Telegram, Slack, dll
fi
```

---

## Error yang Mungkin Muncul

### Pengelolaan MFA

- Browser yang diingat hanya melewati MFA login, bukan pengelolaan authenticator.
- Untuk akun dengan MFA aktif, kirim kode authenticator saat ini ke `POST /api/auth/mfa/verify` dengan body `{"code":"123456"}` dan token sesi biasa.
- Respons `management_ticket` berlaku lima menit. Kirim melalui header `X-MFA-Management` bersama token sesi pada `POST /api/auth/mfa/setup` dan `POST /api/auth/mfa/enable`. Tiket ini bukan token login.
- Setup menghasilkan rahasia baru, tidak menampilkan rahasia lama. Konfirmasi kode authenticator baru melalui endpoint enable. MFA lama tetap aktif jika dibatalkan atau kode salah.
- Penggantian berhasil membatalkan sesi dan kepercayaan browser lama melalui perubahan status kredensial. Gunakan token sesi baru dari respons enable.
- Tiket hilang dari UI saat modal ditutup; membuka menu kembali meminta kode ulang. Tiket kedaluwarsa memerlukan verifikasi ulang.
- Belum mendukung beberapa authenticator independen atau recovery code. Kehilangan HP tetap melalui reset admin.

### Status HTTP

| HTTP Status | Artinya |
|-------------|---------|
| 200 | Sukses |
| 400 | Request salah (parameter kurang/salah format) |
| 401 | Token salah atau expired — login lagi |
| 404 | Device tidak ditemukan |
| 405 | Method salah (misalnya GET ke endpoint yang cuma terima POST) |
| 500 | Error di server — cek log server |
