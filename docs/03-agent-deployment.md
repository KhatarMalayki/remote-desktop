# Cara Pasang Agent di Komputer

Agent itu program kecil (~8 MB) yang kamu pasang di setiap komputer yang mau dikelola.
Agent-nya cuma **kirim info hardware ke server** — ringan, tidak makan resource.

---

## Yang Kamu Butuhkan Sebelum Mulai

1. ✅ Server RemoteDesk sudah jalan (lihat [Cara Install](01-installation.md))
2. ✅ Kamu tahu **IP address** atau **domain** server
3. ✅ Kamu tahu **API Key** server (muncul di log waktu server start)
4. ✅ File `rd-agent` yang sudah di-build untuk OS yang sesuai

---

## Pasang di Windows

### Cara 1: Langsung Jalankan (Paling Cepat)

1. Copy file `rd-agent.exe` ke komputer Windows (taruh di mana saja, misal `C:\RemoteDesk\`)

2. Buka **Command Prompt** atau **PowerShell**

3. Jalankan:
   ```
   C:\RemoteDesk\rd-agent.exe -server http://IP-VPS-KAMU:8080 -key KUNCI-API-KAMU
   ```

4. Kalau muncul tulisan `[agent] connected` dan `[agent] registered` berarti **sukses**!

5. Buka dashboard di browser → device kamu harusnya sudah muncul

> **Catatan:** Kalau kamu tutup Command Prompt, agent-nya ikut mati.
> Untuk biar jalan terus, lihat Cara 2 atau Cara 3.

### Cara 2: Pakai Task Scheduler (Jalan Otomatis Waktu Startup)

Supaya agent jalan otomatis setiap komputer dinyalakan:

1. Buka **Task Scheduler** (ketik "Task Scheduler" di Start Menu)
2. Klik **Create Basic Task...**
3. Isi nama: `RemoteDesk Agent`
4. Trigger: pilih **When the computer starts**
5. Action: pilih **Start a program**
6. Program: browse ke `C:\RemoteDesk\rd-agent.exe`
7. Arguments: `-server http://IP-VPS:8080 -key KUNCI-API`
8. Centang **Open the Properties dialog** → Finish
9. Di Properties, centang **Run whether user is logged on or not**
10. Klik OK, masukkan password admin Windows

Atau kalau mau pakai PowerShell (lebih cepat):

```powershell
$action = New-ScheduledTaskAction -Execute "C:\RemoteDesk\rd-agent.exe" `
  -Argument "-server http://IP-VPS:8080 -key KUNCI-API"
$trigger = New-ScheduledTaskTrigger -AtStartup
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -RunLevel Highest
Register-ScheduledTask -TaskName "RemoteDeskAgent" `
  -Action $action -Trigger $trigger -Principal $principal
```

### Cara 3: Pakai NSSM (Jadi Windows Service)

NSSM adalah tool gratis untuk bikin program apapun jadi Windows Service.

1. Download NSSM dari https://nssm.cc/download
2. Extract, buka Command Prompt di folder NSSM
3. Jalankan:
   ```
   nssm install RemoteDeskAgent C:\RemoteDesk\rd-agent.exe
   ```
4. Di GUI yang muncul:
   - Path: `C:\RemoteDesk\rd-agent.exe`
   - Arguments: `-server http://IP-VPS:8080 -key KUNCI-API`
   - Startup type: Automatic
5. Klik **Install service**
6. Start service:
   ```
   nssm start RemoteDeskAgent
   ```

Sekarang agent jalan sebagai Windows Service — otomatis start, otomatis restart kalau crash.

---

## Pasang di Linux

### Cara 1: Langsung Jalankan (Tes Dulu)

```bash
# 1. Beri izin execute
chmod +x rd-agent

# 2. Jalankan
./rd-agent -server http://IP-VPS:8080 -key KUNCI-API

# Kalau muncul "[agent] connected" dan "[agent] registered" = sukses!
```

Tekan Ctrl+C untuk berhentikan.

### Cara 2: Jalan di Background (Sederhana)

```bash
# Jalan di background, log ke file
nohup ./rd-agent -server http://IP-VPS:8080 -key KUNCI-API > /var/log/rd-agent.log 2>&1 &

# Cek apakah jalan
ps aux | grep rd-agent

# Lihat log
tail -f /var/log/rd-agent.log
```

### Cara 3: Systemd Service (Paling Benar, Auto Restart)

```bash
# 1. Copy binary ke lokasi yang proper
sudo cp rd-agent /usr/local/bin/rd-agent
sudo chmod +x /usr/local/bin/rd-agent

# 2. Buat file service
sudo nano /etc/systemd/system/rd-agent.service
```

Isi file-nya (copy-paste, ganti IP dan API key):

```ini
[Unit]
Description=RemoteDesk Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/rd-agent -server http://IP-VPS:8080 -key KUNCI-API
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Simpan (Ctrl+O, Enter, Ctrl+X), lalu:

```bash
# 3. Aktifkan dan jalankan
sudo systemctl daemon-reload
sudo systemctl enable rd-agent    # supaya jalan otomatis waktu boot
sudo systemctl start rd-agent     # jalankan sekarang

# 4. Cek status
sudo systemctl status rd-agent
# Harus muncul "active (running)"

# 5. Lihat log
journalctl -u rd-agent -f
```

### Cara 4: Pakai Config File

Kalau tidak mau tulis semua parameter di command line:

```bash
# 1. Buat file config
sudo mkdir -p /etc/remotedesk
sudo nano /etc/remotedesk/agent.json
```

Isi:
```json
{
  "server_url": "http://IP-VPS:8080",
  "api_key": "KUNCI-API",
  "device_id": "",
  "heartbeat_seconds": 30
}
```

```bash
# 2. Jalankan pakai config
./rd-agent -config /etc/remotedesk/agent.json
```

Di systemd service, ganti ExecStart jadi:
```
ExecStart=/usr/local/bin/rd-agent -config /etc/remotedesk/agent.json
```

---

## Pasang di macOS

### Langsung Jalankan

```bash
chmod +x rd-agent-darwin-arm64   # untuk Apple Silicon (M1/M2/M3)
# atau
chmod +x rd-agent-darwin-amd64   # untuk Intel Mac

./rd-agent-darwin-arm64 -server http://IP-VPS:8080 -key KUNCI-API
```

### Auto-Start (LaunchDaemon)

```bash
# 1. Copy binary
sudo cp rd-agent-darwin-arm64 /usr/local/bin/rd-agent

# 2. Buat file LaunchDaemon
sudo nano /Library/LaunchDaemons/com.remotedesk.agent.plist
```

Isi:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.remotedesk.agent</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/rd-agent</string>
    <string>-server</string>
    <string>http://IP-VPS:8080</string>
    <string>-key</string>
    <string>KUNCI-API</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
```

```bash
sudo launchctl load /Library/LaunchDaemons/com.remotedesk.agent.plist
```

---

## Deploy ke Banyak Komputer Sekaligus

Kalau kamu punya 50+ server Linux dan mau pasang agent sekaligus:

```bash
#!/bin/bash
# Simpan sebagai deploy-agents.sh

SERVERS="192.168.1.10 192.168.1.11 192.168.1.12 192.168.1.13"
API_KEY="kunci-rahasia-kamu"
SERVER_URL="http://IP-VPS:8080"

for host in $SERVERS; do
  echo "=== Deploying ke $host ==="

  # Copy file agent
  scp rd-agent root@$host:/usr/local/bin/rd-agent

  # Set permission dan jalankan
  ssh root@$host "chmod +x /usr/local/bin/rd-agent && \
    nohup /usr/local/bin/rd-agent \
    -server $SERVER_URL \
    -key $API_KEY \
    > /var/log/rd-agent.log 2>&1 &"

  echo "    Selesai!"
done

echo ""
echo "Semua agent sudah di-deploy! Cek dashboard."
```

```bash
chmod +x deploy-agents.sh
./deploy-agents.sh
```

---

## Penjelasan Parameter Agent

| Parameter | Wajib? | Contoh | Penjelasan |
|-----------|--------|--------|------------|
| `-server` | ✅ Ya | `http://103.15.20.5:8080` | Alamat server RemoteDesk kamu |
| `-key` | ✅ Ya | `abc123secret` | API key yang sama dengan server |
| `-id` | Tidak | `server-prod-01` | ID custom untuk device ini. Kalau kosong, otomatis dibuat |
| `-heartbeat` | Tidak | `30` | Kirim update setiap berapa detik (default: 30) |
| `-config` | Tidak | `/etc/remotedesk/agent.json` | Pakai file config (menggantikan semua flag) |

Bisa juga pakai environment variable:

| Variable | Sama dengan |
|----------|-------------|
| `RD_SERVER_URL` | `-server` |
| `RD_API_KEY` | `-key` |
| `RD_DEVICE_ID` | `-id` |

---

## FAQ Agent

**Q: Agent makan berapa resource?**
A: Sangat kecil. RAM ~3 MB, CPU ~0%. Hampir tidak terasa.

**Q: Aman gak?**
A: Agent cuma kirim info hardware (nama, CPU, RAM, disk, IP). Tidak bisa akses data file kamu.

**Q: Kalau internet mati?**
A: Agent otomatis reconnect setiap beberapa detik. Begitu internet balik, langsung terhubung lagi.

**Q: Bisa ganti device ID?**
A: Bisa, pakai flag `-id`. Atau hapus file `/tmp/rd-device-id` dan restart agent.

**Q: Heartbeat 30 detik, bisa diganti?**
A: Bisa, pakai flag `-heartbeat 60` (untuk 60 detik misalnya). Makin lama = makin hemat bandwidth.
