package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/remote-desktop/internal/vpn"
	"github.com/user/remote-desktop/internal/versioncmp"
)

type vpnSession struct {
	status vpn.Status
	started time.Time
	config vpn.Config
	provisioned bool
}

type vpnServer struct {
	sync.Mutex
	ready bool
	detail string
	network, endpoint, publicKey string
	sessions map[string]*vpnSession
	run func(string,...string)([]byte,error)
}

func vpnExec(name string, args ...string) ([]byte,error) {
	ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second); defer cancel()
	output,err:=exec.CommandContext(ctx,name,args...).Output()
	if err!=nil { return nil,fmt.Errorf("%s gagal; periksa prasyarat VPN pada server",name) }
	return output,nil
}

func (s *Server) initVPN() {
	pilot:=&vpnServer{detail:"Pilot nonaktif; konfigurasi NAS diperlukan. Tidak ada koneksi otomatis.",sessions:map[string]*vpnSession{},run:vpnExec}
	s.vpnPilot=pilot
	if os.Getenv("RD_VPN_PILOT")!="true" { return }
	if err:=s.startVPNHub(); err!=nil { pilot.detail=err.Error(); return }
	pilot.ready=true; pilot.detail="Hub siap; maksimal 2 client, sesi 15 menit, tanpa DNS/default route/advertise LAN."
	go func(){
		for range time.NewTicker(10*time.Second).C {
			pilot.Lock()
			for device,session:=range pilot.sessions {
				if session.status.State=="disconnected" || session.status.State=="error" { continue }
				if time.Since(session.started)>vpn.SessionLimit || time.Now().Unix()-session.status.Updated>int64(vpn.Lease.Seconds()) {
					s.stopVPNSession(device,session,"Lease pilot habis atau agent tidak merespons")
				}
			}
			pilot.Unlock()
		}
	}()
}

func (s *Server) startVPNHub() (err error) {
	pilot:=s.vpnPilot
	if runtime.GOOS!="linux" { return fmt.Errorf("hub pilot memerlukan Linux/TUN; client tidak diaktifkan") }
	pilot.network=os.Getenv("RD_VPN_CIDR"); pilot.endpoint=os.Getenv("RD_VPN_ENDPOINT")
	prefix,err:=vpn.Network(pilot.network); if err!=nil { return err }
	private,public,err:=vpn.NewKey(); if err!=nil { return err }
	probe:=vpn.Config{Network:pilot.network,Endpoint:pilot.endpoint,Address:prefix.Addr().Next().Next().String(),ServerKey:public}
	if err=probe.Validate(); err!=nil { return err }
	forward,err:=os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err!=nil || strings.TrimSpace(string(forward))!="1" { return fmt.Errorf("IPv4 forwarding namespace container belum aktif; gunakan konfigurasi pilot terpisah yang opt-in") }
	if _,err=os.Stat("/dev/net/tun"); err!=nil { return fmt.Errorf("/dev/net/tun belum tersedia; jangan aktifkan client") }
	for _,name:=range []string{"ip","wg","wireguard-go","iptables"} { if _,err=exec.LookPath(name);err!=nil { return fmt.Errorf("runtime hub %s belum tersedia",name) } }
	routeData,err:=pilot.run("ip","-j","-4","route","show","table","all");if err!=nil{return err}
	var routes []struct { Dst string `json:"dst"` }
	if json.Unmarshal(routeData,&routes)!=nil { return fmt.Errorf("rute hub tidak dapat diperiksa") }
	var prefixes []string
	for _,route:=range routes { if route.Dst!="" && route.Dst!="default" { if !strings.Contains(route.Dst,"/"){route.Dst+="/32"}; prefixes=append(prefixes,route.Dst) } }
	if err=vpn.CheckRoutes(pilot.network,prefixes);err!=nil{return err}
	host,_,_:=net.SplitHostPort(pilot.endpoint)
	resolveContext,cancel:=context.WithTimeout(context.Background(),5*time.Second); defer cancel()
	addresses,err:=net.DefaultResolver.LookupNetIP(resolveContext,"ip4",host)
	if err!=nil || len(addresses)==0 { return fmt.Errorf("endpoint hub belum dapat di-resolve") }
	for _,address:=range addresses { if prefix.Contains(address){return fmt.Errorf("endpoint bentrok dengan subnet pilot")} }
	if _,err=pilot.run("ip","link","show","rdpilot");err==nil { return fmt.Errorf("interface rdpilot sudah ada; tidak ditimpa") }
	directory:=filepath.Join(filepath.Dir(s.cfg.DBPath),"vpn-pilot")
	if err=os.MkdirAll(directory,0700);err!=nil{return err}
	keyPath:=filepath.Join(directory,"hub.key")
	if existing,readErr:=os.ReadFile(keyPath);readErr==nil {
		private=strings.TrimSpace(string(existing)); if !vpn.ValidKey(private){return fmt.Errorf("private key hub rusak; tidak diganti otomatis")}
	} else if !os.IsNotExist(readErr) { return readErr } else if err=os.WriteFile(keyPath,[]byte(private),0600);err!=nil{return err}
	if err=os.Chmod(keyPath,0600);err!=nil{return err}
	_,port,_:=net.SplitHostPort(pilot.endpoint)
	if _,err=pilot.run("wireguard-go","rdpilot");err!=nil{return err}
	defer func(){if err!=nil{_,_=pilot.run("ip","link","delete","rdpilot")}}()
	if _,err=pilot.run("wg","set","rdpilot","private-key",keyPath,"listen-port",port);err!=nil{return err}
	if _,err=pilot.run("ip","address","add",prefix.Addr().Next().String()+"/24","dev","rdpilot");err!=nil{return err}
	if _,err=pilot.run("ip","link","set","dev","rdpilot","mtu","1380","up");err!=nil{return err}
	if _,err=pilot.run("iptables","-I","FORWARD","1","-i","rdpilot","!","-o","rdpilot","-j","DROP");err!=nil{return err}
	if _,err=pilot.run("iptables","-I","FORWARD","1","-o","rdpilot","!","-i","rdpilot","-j","DROP");err!=nil{return err}
	if _,err=pilot.run("iptables","-I","FORWARD","1","-i","rdpilot","-o","rdpilot","-j","ACCEPT");err!=nil{return err}
	key,err:=pilot.run("wg","show","rdpilot","public-key");if err!=nil{return err}
	pilot.publicKey=strings.TrimSpace(string(key)); if !vpn.ValidKey(pilot.publicKey){return fmt.Errorf("hub public key tidak valid")}
	_,err=s.db.db.Exec(`CREATE TABLE IF NOT EXISTS vpn_pilot_addresses(device TEXT PRIMARY KEY, slot INTEGER UNIQUE NOT NULL CHECK(slot BETWEEN 2 AND 254))`)
	return err
}

func (s *Server) handleVPN(w http.ResponseWriter,r *http.Request) {
	if getClaims(r).Role!="admin" {jsonError(w,"VPN pilot hanya untuk admin",403);return}
	pilot:=s.vpnPilot
	if pilot==nil {jsonError(w,"VPN pilot belum tersedia",503);return}
	pilot.Lock();defer pilot.Unlock()
	if r.Method==http.MethodGet {
		states:=map[string]vpn.Status{}
		for device,session:=range pilot.sessions {states[device]=session.status}
		jsonResp(w,map[string]interface{}{"ready":pilot.ready,"detail":pilot.detail,"network":pilot.network,"sessions":states},200);return
	}
	if r.Method!=http.MethodPost {http.Error(w,"method not allowed",405);return}
	var input struct {Device string `json:"device"`; Operation string `json:"operation"`}
	r.Body=http.MaxBytesReader(w,r.Body,2048);decoder:=json.NewDecoder(r.Body);decoder.DisallowUnknownFields()
	if decoder.Decode(&input)!=nil || (input.Operation!="connect" && input.Operation!="disconnect") {jsonError(w,"operasi tidak valid",400);return}
	if input.Operation=="disconnect" {
		if session:=pilot.sessions[input.Device];session!=nil{s.stopVPNSession(input.Device,session,"Disconnect diminta admin")}
		jsonResp(w,map[string]string{"status":"disconnect_requested"},202);return
	}
	if !pilot.ready {jsonError(w,pilot.detail,409);return}
	device,err:=s.db.GetDevice(input.Device)
	if err!=nil {jsonError(w,"device tidak ditemukan",404);return}
	if device.OS!="windows" || versioncmp.IsNewer("0.2.61",device.Version) {jsonError(w,"pilot memerlukan Windows agent >=0.2.61",409);return}
	if !s.hub.IsOnline(input.Device) {jsonError(w,"agent offline; perintah tidak diantrekan",409);return}
	active:=0
	for id,session:=range pilot.sessions {
		if session.status.State!="disconnected" && session.status.State!="error" {active++;if id==input.Device{jsonError(w,"sesi masih aktif; disconnect dahulu",409);return}}
	}
	if active>=2 {jsonError(w,"pilot dibatasi dua client bersamaan",409);return}
	var slot int
	err=s.db.db.QueryRow(`SELECT slot FROM vpn_pilot_addresses WHERE device=?`,input.Device).Scan(&slot)
	if err!=nil {
		for slot=2;slot<=254;slot++ {var count int; if err=s.db.db.QueryRow(`SELECT COUNT(*) FROM vpn_pilot_addresses WHERE slot=?`,slot).Scan(&count);err!=nil{jsonError(w,"alokasi IP gagal",500);return};if count==0{break}}
		if slot>254 {jsonError(w,"pool IP habis",409);return}
		if _,err=s.db.db.Exec(`INSERT INTO vpn_pilot_addresses(device,slot) VALUES(?,?)`,input.Device,slot);err!=nil{jsonError(w,"alokasi IP gagal",500);return}
	}
	prefix,_:=vpn.Network(pilot.network);address:=prefix.Addr();for index:=0;index<slot;index++ {address=address.Next()}
	var nonce [16]byte;if _,err=rand.Read(nonce[:]);err!=nil{jsonError(w,"gagal membuat sesi",500);return}
	id:=hex.EncodeToString(nonce[:])
	session:=&vpnSession{started:time.Now(),config:vpn.Config{Address:address.String(),Network:pilot.network,Endpoint:pilot.endpoint,ServerKey:pilot.publicKey},status:vpn.Status{ID:id,State:"preparing",Address:address.String(),Detail:"Menunggu pemeriksaan agent; belum terhubung",Updated:time.Now().Unix()}}
	pilot.sessions[input.Device]=session
	if !s.sendVPN(input.Device,vpn.Command{Operation:"prepare",ID:id,Expires:time.Now().Add(time.Minute).Unix()}) {session.status.State="error";session.status.Detail="agent tidak menerima perintah";jsonError(w,session.status.Detail,409);return}
	_ = s.db.AddLog(input.Device,"vpn_connect_requested",getClaims(r).Username+": pilot manual 15 menit")
	jsonResp(w,session.status,202)
}

func (s *Server) sendVPN(device string,command vpn.Command) bool {
	raw,_:=json.Marshal(map[string]interface{}{"action":"vpn_pilot","data":command});return s.hub.SendToAgent(device,raw)
}

func (s *Server) stopVPNSession(device string,session *vpnSession,reason string) {
	if session.provisioned {
		if _,err:=s.vpnPilot.run("wg","set","rdpilot","peer",session.status.PublicKey,"remove");err!=nil {session.status.State="error";session.status.Detail="Gagal mencabut peer hub; periksa NAS";return}
		session.provisioned=false
	}
	s.sendVPN(device,vpn.Command{Operation:"disconnect",ID:session.status.ID,Expires:time.Now().Add(time.Minute).Unix()})
	session.status.State="disconnecting";session.status.Detail=reason+"; menunggu konfirmasi agent";session.status.Updated=time.Now().Unix()
}

func (s *Server) receiveVPN(device string,raw json.RawMessage) {
	pilot:=s.vpnPilot;if pilot==nil{return}
	var report vpn.Status;if json.Unmarshal(raw,&report)!=nil{return}
	pilot.Lock();defer pilot.Unlock()
	session:=pilot.sessions[device]
	if session==nil || session.status.ID!=report.ID {return}
	if report.State=="error" || report.State=="disconnected" {
		if session.provisioned {s.stopVPNSession(device,session,"agent menghentikan VPN");if session.provisioned{return}}
		session.status.State=report.State;session.status.Detail=report.Detail;session.status.Updated=time.Now().Unix();return
	}
	if session.status.State=="disconnecting" || session.status.State=="disconnected" || session.status.State=="error" {return}
	if report.State=="ready" && session.status.State=="preparing" && vpn.ValidKey(report.PublicKey) {
		for other,peer:=range pilot.sessions {if other!=device && peer.provisioned && peer.status.PublicKey==report.PublicKey{return}}
		session.status.PublicKey=report.PublicKey
		if _,err:=pilot.run("wg","set","rdpilot","peer",report.PublicKey,"allowed-ips",session.config.Address+"/32");err!=nil{session.status.State="error";session.status.Detail=err.Error();return}
		session.provisioned=true;session.status.State="connecting";session.status.Updated=time.Now().Unix()
		if !s.sendVPN(device,vpn.Command{Operation:"connect",ID:report.ID,Expires:time.Now().Add(time.Minute).Unix(),Config:&session.config}) {s.stopVPNSession(device,session,"pengiriman konfigurasi gagal")}
		return
	}
	if report.State!="running" || !session.provisioned{return}
	session.status.Updated=time.Now().Unix()
	if time.Since(session.started)>vpn.SessionLimit {s.stopVPNSession(device,session,"batas sesi 15 menit");return}
	output,err:=pilot.run("wg","show","rdpilot","latest-handshakes")
	hasValidHandshake:=false
	if err==nil {
		for _,line:=range strings.Split(string(output),"\n") {
			fields:=strings.Fields(line);if len(fields)!=2 || fields[0]!=session.status.PublicKey{continue}
			stamp,_:=strconv.ParseInt(fields[1],10,64)
			if stamp>=session.started.Unix() && time.Now().Unix()-stamp<=300 {
				hasValidHandshake=true
				break
			}
		}
	}
	if hasValidHandshake {
		session.status.State="connected"
		session.status.Detail="Handshake WireGuard terverifikasi; split tunnel pilot aktif"
	}
	s.sendVPN(device,vpn.Command{Operation:"keepalive",ID:report.ID,Expires:time.Now().Add(60*time.Second).Unix()})
}
