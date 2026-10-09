package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/user/remote-desktop/internal/vpn"
)

type vpnClient struct {
	sync.Mutex
	status vpn.Status
	private string
	started time.Time
	lastLeaseWrite time.Time
	preparedUntil int64
	seen map[string]bool
}

type vpnLease struct {
	ID string `json:"id"`
	Started time.Time `json:"started"`
	Ack time.Time `json:"ack"`
}

func vpnLeaseSuperseded(initial, current vpnLease, now time.Time) bool {
	return current.ID != initial.ID && vpn.ValidCommand(vpn.Command{ID: current.ID, Expires: now.Add(vpn.Lease).Unix()}, now) &&
		!current.Started.After(current.Ack) && !current.Ack.After(now) && !current.Started.Before(initial.Started) && !vpn.LeaseExpired(now, current.Started, current.Ack)
}

func (a *Agent) initVPNPilot() {
	a.vpnPilot=&vpnClient{status:vpn.Status{State:"disconnected",Detail:"VPN nonaktif; aktivasi manual diperlukan"},seen:map[string]bool{}}
	if err:=vpnPlatformDisconnect();err!=nil{a.vpnPilot.status.State="error";a.vpnPilot.status.Detail=err.Error()}
	initVPNTray()
	go func(){
		for range time.NewTicker(3*time.Second).C {
			a.vpnPilot.Lock()
			if a.vpnPilot.status.State=="running" {
				running,err:=vpnPlatformRunning()
				if err!=nil{
					log.Printf("[vpn] query status failed: %v", err)
					a.vpnPilot.status.State="error";a.vpnPilot.status.Detail=err.Error()
				} else if !running{
					log.Printf("[vpn] service stopped unexpectedly (watchdog or external)")
					a.vpnPilot.status.State="disconnected";a.vpnPilot.status.Detail="VPN dihentikan lokal/watchdog; tidak tersambung ulang otomatis"
				}
			}
			a.reportVPNLocked()
			a.vpnPilot.Unlock()
		}
	}()
}

func (a *Agent) reportVPNLocked() {
	a.vpnPilot.status.Updated=time.Now().Unix()
	raw,_:=json.Marshal(map[string]interface{}{"action":"vpn_pilot_status","data":a.vpnPilot.status})
	_ = a.writeTextMessage(raw)
}

func (a *Agent) handleVPNPilot(command vpn.Command) {
	if a.vpnPilot==nil || !vpn.ValidCommand(command,time.Now()) {return}
	pilot:=a.vpnPilot;pilot.Lock();defer pilot.Unlock()
	fail:=func(err error){pilot.status.State="error";pilot.status.Detail=err.Error()}
	switch command.Operation {
	case "prepare":
		if pilot.seen[command.ID] || pilot.status.State=="running" || pilot.status.State=="ready" {return}
		a.updatingMu.Lock();updating:=a.isUpdating;a.updatingMu.Unlock();if updating{return}
		if len(pilot.seen)>=1000 {fail(fmt.Errorf("batas sesi proses tercapai; restart agent sebelum pilot berikutnya"));break}
		pilot.seen[command.ID]=true;pilot.status=vpn.Status{ID:command.ID,State:"preparing"};pilot.preparedUntil=command.Expires
		private,public,err:=vpnPlatformPrepare(a.cfg.ServerURL, a.cfg.APIKey)
		if err!=nil {fail(err);break}
		pilot.private=private;pilot.status.PublicKey=public;pilot.status.State="ready";pilot.status.Detail="Prasyarat lokal lolos; menunggu konfigurasi hub"
	case "connect":
		if pilot.status.ID!=command.ID || pilot.status.State!="ready" || time.Now().Unix()>pilot.preparedUntil || command.Config==nil {return}
		if err:=command.Config.Validate();err!=nil{fail(err);break}
		pilot.started=time.Now()
		pilot.lastLeaseWrite=pilot.started
		if err:=vpnPlatformConnect(*command.Config,pilot.private,a.cfg.ServerURL,vpnLease{ID:command.ID,Started:pilot.started,Ack:pilot.started});err!=nil{fail(err);break}
		log.Printf("[vpn] tunnel connected as %s", command.Config.Address)
		pilot.status.State="running";pilot.status.Address=command.Config.Address;pilot.status.Detail="Tunnel berjalan; menunggu handshake hub dan lease kontrol"
	case "keepalive":
		if pilot.status.ID!=command.ID || pilot.status.State!="running"{return}
		if err:=vpnWriteLease(vpnLease{ID:command.ID,Started:pilot.started,Ack:time.Now()});err!=nil {
			log.Printf("[vpn] lease write failed: %v", err)
			if time.Since(pilot.lastLeaseWrite) > vpn.Lease-15*time.Second {
				stopErr:=vpnPlatformDisconnect();fail(fmt.Errorf("penyimpanan lease gagal berturut-turut: %v; hasil disconnect: %v",err,stopErr))
			}
		} else {
			pilot.lastLeaseWrite=time.Now()
		}
	case "disconnect":
		if pilot.status.ID!=command.ID{return}
		log.Printf("[vpn] disconnect command received from server")
		if err:=vpnPlatformDisconnect();err!=nil{fail(err)}else{pilot.status.State="disconnected";pilot.status.Detail="VPN nonaktif; service RemoteDesk tetap berjalan"}
	default:return
	}
	a.reportVPNLocked()
}

func (a *Agent) stopVPNForUpdate() error {
	if a.vpnPilot==nil{return nil}
	a.vpnPilot.Lock();defer a.vpnPilot.Unlock()
	if err:=vpnPlatformDisconnect();err!=nil{return err}
	a.vpnPilot.status.State="disconnected";a.vpnPilot.status.Detail="VPN dimatikan sebelum update agent";a.reportVPNLocked()
	return nil
}
