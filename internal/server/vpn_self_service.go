package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/user/remote-desktop/internal/vpn"
)

type vpnSelfPolicy struct {
	Enabled  bool   `json:"enabled"`
	RouteLAN string `json:"route_lan"`
}

func (s *Server) vpnSelfPolicy(device string) vpnSelfPolicy {
	var policy vpnSelfPolicy
	if json.Unmarshal([]byte(s.db.GetSystemSetting("vpn_self_service:"+device)), &policy) != nil {
		return vpnSelfPolicy{}
	}
	return policy
}

func (s *Server) handleVPNSelfPolicy(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role != "admin" {
		jsonError(w, "Policy VPN hanya untuk admin", 403)
		return
	}
	if s.vpnPilot == nil {
		jsonError(w, "VPN belum tersedia", 503)
		return
	}
	deviceID := r.URL.Query().Get("device")
	device, err := s.db.GetDevice(deviceID)
	if err != nil {
		jsonError(w, "Device tidak ditemukan", 404)
		return
	}
	s.vpnPilot.Lock()
	defer s.vpnPilot.Unlock()
	if r.Method == http.MethodGet {
		jsonResp(w, s.vpnSelfPolicy(deviceID), 200)
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", 405)
		return
	}
	var policy vpnSelfPolicy
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&policy) != nil || decoder.Decode(new(any)) != io.EOF {
		jsonError(w, "Policy tidak valid", 400)
		return
	}
	if policy.Enabled && device.OS != "windows" {
		jsonError(w, "Self-service memerlukan Windows", 409)
		return
	}
	if policy.RouteLAN != "" {
		if _, err := vpn.LANPrefix(policy.RouteLAN); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
	}
	encoded, _ := json.Marshal(policy)
	if err := s.db.SetSystemSetting("vpn_self_service:"+deviceID, string(encoded)); err != nil {
		jsonError(w, "Policy gagal disimpan", 500)
		return
	}
	if session := s.vpnPilot.sessions[deviceID]; session != nil && session.selfService && vpnSessionActive(session) {
		s.stopVPNSession(deviceID, session, "Policy self-service diubah admin; connect ulang hanya jika diizinkan")
	}
	_ = s.db.AddLog(deviceID, "vpn_self_policy", getClaims(r).Username+": "+string(encoded))
	jsonResp(w, policy, 200)
}

func (s *Server) selfVPNAllowed(device string, session *vpnSession) bool {
	policy := s.vpnSelfPolicy(device)
	return policy.Enabled && ((policy.RouteLAN == "" && len(session.config.Routes) == 0) || (len(session.config.Routes) == 1 && session.config.Routes[0] == policy.RouteLAN))
}

func (s *Server) receiveVPNSelfRequest(device, operation string, raw json.RawMessage) {
	if operation != "connect" && operation != "disconnect" {
		return
	}
	var input struct {
		ID string `json:"id"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if len(raw) > 128 || decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF || !vpn.ValidCommand(vpn.Command{ID: input.ID, Expires: time.Now().Add(time.Second).Unix()}, time.Now()) {
		return
	}
	reply := vpn.SelfReply{ID: input.ID, Detail: "VPN belum tersedia"}
	send := func() {
		data, _ := json.Marshal(map[string]any{"action": "vpn_self_result", "data": reply})
		s.hub.SendToAgent(device, data)
	}
	if s.vpnPilot == nil {
		send()
		return
	}
	pilot := s.vpnPilot
	pilot.Lock()
	defer pilot.Unlock()
	if operation == "disconnect" {
		if session := pilot.sessions[device]; session != nil && vpnSessionActive(session) {
			s.stopVPNSession(device, session, "Disconnect diminta pengguna lokal")
		}
		reply.Accepted = true
		reply.Detail = "Disconnect diminta; persiapan dan akses sesi dihentikan."
		_ = s.db.AddLog(device, "vpn_self_disconnect", reply.Detail)
		send()
		return
	}
	if pilot.selfRequests == nil {
		pilot.selfRequests = make(map[string]time.Time)
	}
	if _, err := s.db.GetDevice(device); err != nil {
		reply.Detail = "Perangkat belum terdaftar"
		send()
		return
	}
	if time.Since(pilot.selfRequests[device]) < 5*time.Second {
		reply.Detail = "Tunggu 5 detik sebelum mencoba lagi"
		send()
		return
	}
	pilot.selfRequests[device] = time.Now()
	policy := s.vpnSelfPolicy(device)
	if !policy.Enabled {
		reply.Detail = "Connect mandiri belum diizinkan admin untuk perangkat ini"
	} else {
		_, _, err := s.startVPNConnection(vpnConnectRequest{Device: device, RouteLAN: policy.RouteLAN}, "tray perangkat "+device)
		if err != nil {
			reply.Detail = err.Error()
		} else {
			pilot.sessions[device].selfService = true
			reply.Accepted = true
			reply.Detail = "Disetujui policy admin. Tunnel sedang disiapkan; periksa status VPN di tray."
		}
	}
	_ = s.db.AddLog(device, "vpn_self_connect", reply.Detail)
	send()
}
