package p2p

import (
	"fmt"
	"net"
	"time"

	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/pion/stun/v3"
)

const DefaultSTUNServer = "stun.l.google.com:19302"

func GatherCandidates(stunServer string) (net.PacketConn, []protocol.Candidate, error) {
	if stunServer == "" {
		stunServer = DefaultSTUNServer
	}

	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, nil, fmt.Errorf("listen udp: %w", err)
	}

	var candidates []protocol.Candidate

	localPort := conn.LocalAddr().(*net.UDPAddr).Port
	hostAddrs := localAddresses()
	for _, ip := range hostAddrs {
		candidates = append(candidates, protocol.Candidate{
			IP:   ip,
			Port: localPort,
			Type: "host",
		})
	}

	srflx, err := stunReflexive(conn, stunServer)
	if err == nil {
		candidates = append(candidates, *srflx)
	}

	if len(candidates) == 0 {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("no candidates gathered")
	}

	return conn, candidates, nil
}

func stunReflexive(conn net.PacketConn, server string) (*protocol.Candidate, error) {
	serverAddr, err := net.ResolveUDPAddr("udp4", server)
	if err != nil {
		return nil, err
	}

	msg, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	if _, err := conn.WriteTo(msg.Raw, serverAddr); err != nil {
		_ = conn.SetReadDeadline(time.Time{})
		return nil, err
	}

	buf := make([]byte, 1024)
	n, _, err := conn.ReadFrom(buf)

	_ = conn.SetReadDeadline(time.Time{})

	if err != nil {
		return nil, err
	}

	resp := new(stun.Message)
	resp.Raw = buf[:n]
	if err := resp.Decode(); err != nil {
		return nil, err
	}

	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(resp); err != nil {
		return nil, err
	}

	return &protocol.Candidate{
		IP:   xorAddr.IP.String(),
		Port: xorAddr.Port,
		Type: "srflx",
	}, nil
}

func localAddresses() []string {
	var addrs []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return addrs
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		ifAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range ifAddrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			addrs = append(addrs, ip.String())
		}
	}
	return addrs
}
