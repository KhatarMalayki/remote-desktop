//go:build windows

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/user/remote-desktop/internal/vpn"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const vpnService = "WireGuardTunnel$"+vpn.TunnelName

func vpnDirectory() string { return filepath.Join(os.Getenv("ProgramData"),"RemoteDeskVPN") }
func vpnConfigPath() string { return filepath.Join(vpnDirectory(),vpn.TunnelName+".conf") }

func vpnEmbeddedRuntimePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "wireguard.exe")
}

func vpnSystemRuntimePath() string {
	return filepath.Join(os.Getenv("ProgramFiles"), "WireGuard", "wireguard.exe")
}

func vpnRuntimePath() string {
	if embedded := vpnEmbeddedRuntimePath(); embedded != "" {
		if _, err := os.Stat(embedded); err == nil {
			return embedded
		}
	}
	return vpnSystemRuntimePath()
}

func secureVPNDirectory() error {
	directory:=vpnDirectory()
	if !filepath.IsAbs(directory){return fmt.Errorf("ProgramData tidak valid")}
	descriptor,err:=windows.SecurityDescriptorFromString("O:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)");if err!=nil{return err}
	attributes:=windows.SecurityAttributes{Length:uint32(unsafe.Sizeof(windows.SecurityAttributes{})),SecurityDescriptor:descriptor}
	path,_:=windows.UTF16PtrFromString(directory)
	err=windows.CreateDirectory(path,&attributes)
	if err!=nil && !errors.Is(err,windows.ERROR_ALREADY_EXISTS){return err}
	flags,err:=windows.GetFileAttributes(path);if err!=nil{return err}
	if flags&windows.FILE_ATTRIBUTE_REPARSE_POINT!=0{return fmt.Errorf("direktori VPN berupa reparse point; ditolak")}
	existing,err:=windows.GetNamedSecurityInfo(directory,windows.SE_FILE_OBJECT,windows.OWNER_SECURITY_INFORMATION);if err!=nil{return err}
	owner,_,err:=existing.Owner();if err!=nil{return err}
	if owner.String()!="S-1-5-18" && owner.String()!="S-1-5-32-544"{return fmt.Errorf("direktori VPN bukan milik SYSTEM/Administrators")}
	dacl,_,err:=descriptor.DACL();if err!=nil{return err}
	return windows.SetNamedSecurityInfo(directory,windows.SE_FILE_OBJECT,windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,nil,nil,dacl,nil)
}

func vpnAtomicFile(name string,data []byte) error {
	path:=filepath.Join(vpnDirectory(),name)
	if info,err:=os.Lstat(path);err==nil && info.Mode()&os.ModeSymlink!=0{return fmt.Errorf("file VPN symlink ditolak")}
	file,err:=os.CreateTemp(vpnDirectory(),".vpn-*");if err!=nil{return err}
	temporary:=file.Name();defer os.Remove(temporary)
	if _,err=file.Write(data);err!=nil{file.Close();return err}
	if err=file.Sync();err!=nil{file.Close();return err}
	if err=file.Close();err!=nil{return err}
	deadline:=time.Now().Add(time.Second)
	for {
		err=os.Rename(temporary,path)
		if err==nil || (!errors.Is(err,windows.ERROR_SHARING_VIOLATION) && !errors.Is(err,windows.ERROR_ACCESS_DENIED)) || !time.Now().Before(deadline){return err}
		time.Sleep(25*time.Millisecond)
	}
}

func vpnPowershell(script string,input string)([]byte,error) {
	ctx,cancel:=context.WithTimeout(context.Background(),12*time.Second);defer cancel()
	command:=exec.CommandContext(ctx,"powershell.exe","-NoProfile","-NonInteractive","-Command","$ErrorActionPreference='Stop'; "+script)
	command.SysProcAttr=&syscall.SysProcAttr{HideWindow:true};command.Stdin=strings.NewReader(input)
	output,err:=command.Output();if err!=nil{return nil,fmt.Errorf("pemeriksaan Windows gagal; VPN tidak diaktifkan")};return output,nil
}

func ensureVPNRuntime(serverURL, apiKey string) error {
	if _, err := os.Stat(vpnRuntimePath()); err == nil {
		return nil
	}
	targetPath := vpnEmbeddedRuntimePath()
	if targetPath == "" {
		return fmt.Errorf("direktori agent tidak valid")
	}
	if strings.TrimSpace(serverURL) == "" {
		return fmt.Errorf("runtime WireGuard resmi belum terpasang dan URL server tidak tersedia")
	}
	downloadURL := fmt.Sprintf("%s/api/agent/download?os=windows&arch=amd64&file=wireguard.exe&key=%s",
		strings.TrimRight(serverURL, "/"), url.QueryEscape(apiKey))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("gagal membuat request download wireguard: %w", err)
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal mengunduh runtime WireGuard: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server belum menyediakan runtime WireGuard (HTTP %d)", resp.StatusCode)
	}

	tempPath := targetPath + ".tmp"
	_ = os.Remove(tempPath)
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("gagal menyimpan runtime WireGuard sementara: %w", err)
	}
	n, err := io.Copy(file, resp.Body)
	_ = file.Close()
	if err != nil || n < 1000000 {
		_ = os.Remove(tempPath)
		return fmt.Errorf("download runtime WireGuard rusak atau tidak lengkap")
	}

	output, err := vpnPowershell("$s=Get-AuthenticodeSignature -LiteralPath ([Console]::In.ReadToEnd()); @{status=$s.Status.ToString();subject=$s.SignerCertificate.Subject}|ConvertTo-Json -Compress", tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("verifikasi signature runtime WireGuard gagal: %w", err)
	}
	var signature struct {
		Status  string
		Subject string
	}
	if json.Unmarshal(output, &signature) != nil || signature.Status != "Valid" || !strings.Contains(signature.Subject, "WireGuard") {
		_ = os.Remove(tempPath)
		return fmt.Errorf("signature runtime WireGuard yang diunduh tidak valid; file ditolak")
	}

	_ = os.Remove(targetPath)
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("gagal memindahkan runtime WireGuard: %w", err)
	}
	return nil
}

func vpnPlatformPrepare(serverURL, apiKey string)(string,string,error) {
	if !windows.GetCurrentProcessToken().IsElevated(){return "","",fmt.Errorf("VPN memerlukan agent service SYSTEM/admin")}
	if err:=secureVPNDirectory();err!=nil{return "","",err}
	if err:=ensureVPNRuntime(serverURL, apiKey);err!=nil{return "","",err}
	if _,err:=os.Stat(vpnRuntimePath());err!=nil{return "","",fmt.Errorf("runtime WireGuard resmi belum terpasang; pasang dari wireguard.com/install pada PC pilot, tanpa mengimpor tunnel")}
	output,err:=vpnPowershell("$s=Get-AuthenticodeSignature -LiteralPath ([Console]::In.ReadToEnd()); @{status=$s.Status.ToString();subject=$s.SignerCertificate.Subject}|ConvertTo-Json -Compress",vpnRuntimePath())
	if err!=nil{return "","",err}
	var signature struct {Status string;Subject string}
	if json.Unmarshal(output,&signature)!=nil || signature.Status!="Valid" || !strings.Contains(signature.Subject,"WireGuard") {return "","",fmt.Errorf("signature runtime WireGuard tidak valid; tidak dieksekusi")}
	if err:=vpnPlatformDisconnect();err!=nil{return "","",err}
	return vpn.NewKey()
}

func openVPNService(access uint32)(*mgr.Service,error) {
	manager,err:=windows.OpenSCManager(nil,nil,windows.SC_MANAGER_CONNECT);if err!=nil{return nil,err};defer windows.CloseServiceHandle(manager)
	name,_:=windows.UTF16PtrFromString(vpnService)
	handle,err:=windows.OpenService(manager,name,access|windows.SERVICE_QUERY_CONFIG);if err!=nil{return nil,err}
	service:=&mgr.Service{Name:vpnService,Handle:handle}
	config,err:=service.Config()
	if err!=nil{service.Close();return nil,err}
	arguments,err:=windows.DecomposeCommandLine(config.BinaryPathName)
	if err!=nil || len(arguments)!=3 || !strings.EqualFold(arguments[0],vpnRuntimePath()) || arguments[1]!="/tunnelservice" || !strings.EqualFold(arguments[2],vpnConfigPath()) {
		service.Close();return nil,fmt.Errorf("nama service VPN dipakai instalasi lain; tidak disentuh")
	}
	return service,nil
}

func vpnPlatformRunning()(bool,error) {
	service,err:=openVPNService(windows.SERVICE_QUERY_STATUS)
	if errors.Is(err,windows.ERROR_SERVICE_DOES_NOT_EXIST){return false,nil};if err!=nil{return false,err};defer service.Close()
	status,err:=service.Query();return status.State!=svc.Stopped,err
}

func vpnPlatformDisconnect() error {
	service,err:=openVPNService(windows.SERVICE_STOP|windows.SERVICE_QUERY_STATUS)
	if errors.Is(err,windows.ERROR_SERVICE_DOES_NOT_EXIST){return nil};if err!=nil{return err};defer service.Close()
	status,err:=service.Query();if err!=nil{return err}
	if status.State!=svc.Stopped {
		if _,err=service.Control(svc.Stop);err!=nil && !errors.Is(err,windows.ERROR_SERVICE_NOT_ACTIVE){return fmt.Errorf("service VPN belum berhenti: %w",err)}
		deadline:=time.Now().Add(12*time.Second)
		for time.Now().Before(deadline) {
			status,err=service.Query();if err!=nil{return err};if status.State==svc.Stopped{break};time.Sleep(200*time.Millisecond)
		}
		if status.State!=svc.Stopped{return fmt.Errorf("service VPN belum mengonfirmasi berhenti; gunakan tray atau hub untuk mencabut akses")}
	}
	return nil
}

func vpnWriteLease(lease vpnLease)error {raw,err:=json.Marshal(lease);if err!=nil{return err};return vpnAtomicFile("lease.json",raw)}

func vpnPreflight(config vpn.Config,serverURL string)error {
	if err:=config.Validate();err!=nil{return err}
	output,err:=vpnPowershell("ConvertTo-Json -InputObject @(Get-NetRoute -AddressFamily IPv4 -PolicyStore ActiveStore | Select-Object -ExpandProperty DestinationPrefix) -Compress","")
	if err!=nil{return err}
	var routes []string;if json.Unmarshal(output,&routes)!=nil{return fmt.Errorf("rute Windows tidak dapat diperiksa")}
	if err=vpn.CheckRoutes(config.Network,routes);err!=nil{return err}
	prefix,_:=vpn.Network(config.Network)
	server,err:=url.Parse(serverURL);if err!=nil || server.Hostname()==""{return fmt.Errorf("alamat kontrol RemoteDesk tidak valid")}
	endpoint,_,_:=net.SplitHostPort(config.Endpoint)
	for _,host:=range []string{endpoint,server.Hostname()} {
		ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
		addresses,lookupErr:=net.DefaultResolver.LookupNetIP(ctx,"ip",host);cancel()
		if lookupErr!=nil || len(addresses)==0{return fmt.Errorf("endpoint/kontrol belum dapat di-resolve; VPN dibatalkan")}
		for _,address:=range addresses{if prefix.Contains(address.Unmap()){return fmt.Errorf("VPN akan mengambil alih jalur kontrol/endpoint; ditolak")}}
	}
	interfaces,err:=net.Interfaces();if err!=nil{return err}
	for _,adapter:=range interfaces{
		addresses,err:=adapter.Addrs();if err!=nil{return err}
		for _,address:=range addresses{parsed,err:=netip.ParsePrefix(address.String());if err==nil && prefix.Overlaps(parsed){return fmt.Errorf("subnet bertabrakan dengan interface %s",adapter.Name)}}
	}
	return nil
}

func vpnPlatformConnect(config vpn.Config,private,serverURL string,lease vpnLease)(err error) {
	lock,err:=vpnLockLifecycle(15*time.Second);if err!=nil{return fmt.Errorf("siklus VPN sedang dipakai atau tidak dapat dikunci: %w",err)};defer lock.Close()
	if err=vpnPreflight(config,serverURL);err!=nil{return err}
	rendered,err:=config.Render(private);if err!=nil{return err}
	if err=vpnAtomicFile(vpn.TunnelName+".conf",[]byte(rendered));err!=nil{return err}
	manager,err:=mgr.Connect();if err!=nil{return err};defer manager.Disconnect()
	service,err:=openVPNService(windows.SERVICE_ALL_ACCESS)
	if errors.Is(err,windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		service,err=manager.CreateService(vpnService,vpnRuntimePath(),mgr.Config{DisplayName:"RemoteDesk VPN Pilot",StartType:mgr.StartManual,Dependencies:[]string{"Nsi","TcpIp"}},"/tunnelservice",vpnConfigPath())
	}
	if err!=nil{return err};defer service.Close()
	configuration,err:=service.Config();if err!=nil{return err};configuration.StartType=mgr.StartManual
	if err=service.UpdateConfig(configuration);err!=nil{return err}
	sid:=uint32(windows.SERVICE_SID_TYPE_UNRESTRICTED)
	if err=windows.ChangeServiceConfig2(service.Handle,windows.SERVICE_CONFIG_SERVICE_SID_INFO,(*byte)(unsafe.Pointer(&sid)));err!=nil{return err}
	descriptor,err:=windows.SecurityDescriptorFromString("D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;0x25;;;IU)");if err!=nil{return err}
	dacl,_,err:=descriptor.DACL();if err!=nil{return err}
	if err=windows.SetNamedSecurityInfo(vpnService,windows.SE_SERVICE,windows.DACL_SECURITY_INFORMATION,nil,nil,dacl,nil);err!=nil{return err}
	if err=vpnWriteLease(lease);err!=nil{return err}
	readyPath:=filepath.Join(vpnDirectory(),"watchdog.ready");if err=os.Remove(readyPath);err!=nil && !os.IsNotExist(err){return err}
	executable,err:=os.Executable();if err!=nil{return err}
	watchdog:=exec.Command(executable,"--vpn-watchdog");watchdog.SysProcAttr=&syscall.SysProcAttr{HideWindow:true}
	if err=watchdog.Start();err!=nil{return fmt.Errorf("watchdog gagal dimulai; VPN tidak diaktifkan")}
	go watchdog.Wait()
	defer func(){if err!=nil{if stopErr:=vpnPlatformDisconnect();stopErr!=nil{err=fmt.Errorf("%v; cleanup gagal: %v",err,stopErr)}}}()
	ready:=false
	for deadline:=time.Now().Add(5*time.Second);time.Now().Before(deadline);time.Sleep(100*time.Millisecond){raw,_:=os.ReadFile(readyPath);if string(raw)==lease.ID{ready=true;break}}
	if !ready{return fmt.Errorf("watchdog tidak siap; VPN tidak diaktifkan")}
	if err=service.Start();err!=nil{return err}
	for deadline:=time.Now().Add(12*time.Second);time.Now().Before(deadline);time.Sleep(200*time.Millisecond){status,queryErr:=service.Query();if queryErr!=nil{return queryErr};if status.State==svc.Running{return nil};if status.State==svc.Stopped{return fmt.Errorf("runtime WireGuard berhenti sebelum aktif")}}
	return fmt.Errorf("startup VPN timeout; rollback diminta")
}

var vpnRunningCheck = vpnPlatformRunning
var vpnDisconnectFunc = vpnPlatformDisconnect

func RunVPNWatchdog() error {
	raw,err:=os.ReadFile(filepath.Join(vpnDirectory(),"lease.json"));if err!=nil{return err}
	var initial vpnLease;if json.Unmarshal(raw,&initial)!=nil || len(initial.ID)!=32 || vpn.LeaseExpired(time.Now(),initial.Started,initial.Ack){return fmt.Errorf("lease watchdog tidak valid")}
	if err=vpnAtomicFile("watchdog.ready",[]byte(initial.ID));err!=nil{return err}
	seenRunning:=false
	lastAck:=initial.Ack
	for {
		done,running,nextAck,stepErr:=vpnWatchdogStep(initial,seenRunning,lastAck)
		seenRunning=seenRunning || running
		if !nextAck.IsZero() {lastAck=nextAck}
		if done{return stepErr}
		time.Sleep(2*time.Second)
	}
}

func vpnLockLifecycle(wait time.Duration) (*os.File,error) {
	file,err:=os.OpenFile(filepath.Join(vpnDirectory(),"lifecycle.lock"),os.O_CREATE|os.O_RDWR,0600)
	if err!=nil{return nil,err}
	deadline:=time.Now().Add(wait)
	for {
		err=windows.LockFileEx(windows.Handle(file.Fd()),windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,0,1,0,&windows.Overlapped{})
		if err==nil{return file,nil}
		if !errors.Is(err,windows.ERROR_LOCK_VIOLATION) || !time.Now().Before(deadline){file.Close();return nil,err}
		time.Sleep(50*time.Millisecond)
	}
}

func vpnWatchdogStep(initial vpnLease,seenRunning bool,lastAck time.Time)(bool,bool,time.Time,error) {
	lock,err:=vpnLockLifecycle(0)
	if errors.Is(err,windows.ERROR_LOCK_VIOLATION){return false,false,lastAck,nil}
	if err!=nil {
		if stopErr:=vpnPlatformDisconnect();stopErr!=nil{return false,false,lastAck,stopErr}
		return true,false,lastAck,err
	}
	defer lock.Close()
	raw,readErr:=os.ReadFile(filepath.Join(vpnDirectory(),"lease.json"))
	var lease vpnLease
	decodeErr:=json.Unmarshal(raw,&lease)
	if readErr==nil && decodeErr==nil {
		if vpnLeaseSuperseded(initial,lease,time.Now()) {return true,false,lease.Ack,nil}
		if lease.ID!=initial.ID || lease.Started.Unix()!=initial.Started.Unix() {
			log.Printf("[vpn-watchdog] lease mismatch: id=%s vs %s", lease.ID, initial.ID)
			err=vpnPlatformDisconnect()
			return err==nil,false,lastAck,err
		}
		lastAck=lease.Ack
	}
	running,queryErr:=vpnRunningCheck()
	if queryErr==nil && !running && (seenRunning || time.Since(initial.Started)>20*time.Second){return true,false,lastAck,nil}
	expired:=vpn.LeaseExpired(time.Now(),initial.Started,lastAck)
	if expired || queryErr!=nil {
		log.Printf("[vpn-watchdog] stopping tunnel: expired=%v queryErr=%v lastAckAge=%v", expired, queryErr, time.Since(lastAck))
		err=vpnDisconnectFunc()
		return err==nil,running,lastAck,err
	}
	return false,running,lastAck,nil
}
