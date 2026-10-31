package main

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"time"
)

// LAN discovery: every second, a computer that is hosting broadcasts a small UDP beacon listing its games.
// Every copy listens and shows those games in its Co-op list. Hosts typed in by address are polled over HTTP.

type beacon struct {
	App   string `json:"app"`
	ID    string `json:"id"`
	Port  int    `json:"port"`
	Games []game `json:"games"`
}

func discovery(h *hub) {
	lc := net.ListenConfig{Control: udpControl}
	pc, err := lc.ListenPacket(context.Background(), "udp4", ":"+strconv.Itoa(discPort))
	if err != nil {
		logger.Printf("discovery listen: %v (other games won't show up automatically; join by address still works)", err)
		pc, err = lc.ListenPacket(context.Background(), "udp4", ":0")
		if err != nil {
			logger.Printf("discovery send socket: %v", err)
			pc = nil
		}
	}
	if pc != nil {
		go func() {
			buf := make([]byte, 64<<10)
			for {
				n, src, err := pc.ReadFrom(buf)
				if err != nil {
					time.Sleep(time.Second)
					continue
				}
				var b beacon
				if json.Unmarshal(buf[:n], &b) != nil || b.App != "dz" || b.ID == instanceID || b.Port <= 0 || b.Port > 65535 {
					continue
				}
				ua, ok := src.(*net.UDPAddr)
				if !ok {
					continue
				}
				if len(b.Games) > 16 {
					b.Games = b.Games[:16]
				}
				h.setRemote(net.JoinHostPort(ua.IP.String(), strconv.Itoa(b.Port)), b.Games, false)
			}
		}()
	}
	tick := time.NewTicker(time.Second)
	n := 0
	for range tick.C {
		n++
		if pc != nil {
			if games := h.localGames(); len(games) > 0 {
				pkt, _ := json.Marshal(beacon{App: "dz", ID: instanceID, Port: port, Games: games})
				for _, dst := range broadcastAddrs() {
					pc.WriteTo(pkt, &net.UDPAddr{IP: dst, Port: discPort})
				}
			}
		}
		if n%2 == 0 {
			for _, a := range h.manualHosts() {
				go h.pollHost(a)
			}
		}
		h.refreshVirtual()
	}
}

func broadcastAddrs() []net.IP {
	out := []net.IP{net.IPv4bcast}
	ifs, _ := net.Interfaces()
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagBroadcast == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4, m := ipn.IP.To4(), ipn.Mask
			if ip4 == nil || len(m) < 4 {
				continue
			}
			if len(m) == 16 {
				m = m[12:]
			}
			b := make(net.IP, 4)
			for i := 0; i < 4; i++ {
				b[i] = ip4[i] | ^m[i]
			}
			out = append(out, b)
		}
	}
	return out
}
