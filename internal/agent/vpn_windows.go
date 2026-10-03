//go:build windows

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
	return os.Rename(temporary,path)
}

func vpnPowershell(script string,input string)([]byte,error) {
	ctx,cancel:=context.WithTimeout(context.Background(),12*time.Second);defer cancel()
	command:=exec.CommandContext(ctx,"powershell.exe","-NoProfile","-NonInteractive","-Command","$ErrorActionPreference='Stop'; "+script)
	command.SysProcAttr=&syscall.SysProcAttr{HideWindow:true};command.Stdin=strings.NewReader(input)
	output,err:=command.Output();if err!=nil{return nil,fmt.Errorf("pemeriksaan Windows gagal; VPN tidak diaktifkan")};return output,nil
}

func vpnPlatformPrepare()(string,string,error) {
	if !windows.GetCurrentProcessToken().IsElevated(){return "","",fmt.Errorf("VPN memerlukan agent service SYSTEM/admin")}
	if err:=secureVPNDirectory();err!=nil{return "","",err}
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

func RunVPNWatchdog() error {
	raw,err:=os.ReadFile(filepath.Join(vpnDirectory(),"lease.json"));if err!=nil{return err}
	var initial vpnLease;if json.Unmarshal(raw,&initial)!=nil || len(initial.ID)!=32 || vpn.LeaseExpired(time.Now(),initial.Started,initial.Ack){return fmt.Errorf("lease watchdog tidak valid")}
	if err=vpnAtomicFile("watchdog.ready",[]byte(initial.ID));err!=nil{return err}
	seenRunning:=false
	for {
		running,queryErr:=vpnPlatformRunning()
		if queryErr==nil && running{seenRunning=true}
		if queryErr==nil && !running && (seenRunning || time.Since(initial.Started)>20*time.Second){return nil}
		raw,err=os.ReadFile(filepath.Join(vpnDirectory(),"lease.json"))
		var lease vpnLease
		invalid:=err!=nil || json.Unmarshal(raw,&lease)!=nil || lease.ID!=initial.ID || !lease.Started.Equal(initial.Started) || vpn.LeaseExpired(time.Now(),initial.Started,lease.Ack)
		if invalid || queryErr!=nil {if err=vpnPlatformDisconnect();err==nil{return nil}}
		time.Sleep(2*time.Second)
	}
}
