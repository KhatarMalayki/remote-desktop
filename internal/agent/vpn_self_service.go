package agent

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/user/remote-desktop/internal/vpn"
)

func (a *Agent) requestSelfVPN(operation string) (vpn.SelfReply, error) {
	if operation != "connect" && operation != "disconnect" {
		return vpn.SelfReply{}, fmt.Errorf("operasi tidak diizinkan")
	}
	pilot := a.vpnPilot
	pilot.selfMu.Lock()
	defer pilot.selfMu.Unlock()
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return vpn.SelfReply{}, err
	}
	id := hex.EncodeToString(nonce[:])
	raw, _ := json.Marshal(map[string]any{"action": "vpn_self_" + operation, "data": map[string]string{"id": id}})
	if err := a.writeTextMessage(raw); err != nil {
		return vpn.SelfReply{}, fmt.Errorf("server tidak terhubung: %w", err)
	}
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case reply := <-pilot.selfReplies:
			if reply.ID == id {
				return reply, nil
			}
		case <-timer.C:
			return vpn.SelfReply{}, fmt.Errorf("server belum menjawab; periksa status VPN sebelum mencoba lagi")
		}
	}
}
