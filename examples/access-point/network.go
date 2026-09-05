package main

import (
	"log/slog"
	"net/netip"

	"github.com/soypat/lneto/dhcp/dhcpv4"
	"github.com/soypat/lneto/ethernet"
	"github.com/soypat/lneto/ipv4"
	"github.com/soypat/lneto/x/xnet"
)

var apAddr = netip.MustParseAddr("192.168.4.1")

func configureNetwork(stack *xnet.StackAsync, server *dhcpv4.Server, mac [6]byte, logger *slog.Logger) error {
	addr := apAddr.As4()
	err := stack.Reset(xnet.StackConfig{
		Hostname:            "pico-ap",
		RandSeed:            1,
		StaticAddress4:      addr,
		HardwareAddress:     mac,
		MTU:                 ethernet.MaxMTU,
		MaxActiveUDPPorts:   1,
		PassivePeers:        8,
		ICMPQueueLimit:      2,
		AcceptIPv4Broadcast: true,
		Logger:              logger,
	})
	if err != nil {
		return err
	}
	if err := stack.EnableICMP(true); err != nil {
		return err
	}
	err = server.Configure(dhcpv4.ServerConfig{
		ServerAddr: addr,
		Gateway:    addr,
		Subnet:     ipv4.PrefixFrom(addr, 24),
	})
	if err != nil {
		return err
	}
	return stack.RegisterUDP4(&loggedDHCPServer{Server: server, logger: logger}, [4]byte{}, dhcpv4.DefaultClientPort)
}

// loggedDHCPServer keeps the access-point serial log useful when a client
// cannot complete DHCP. In particular, the transaction ID and client identity
// show whether a REQUEST belongs to a preceding DISCOVER.
type loggedDHCPServer struct {
	*dhcpv4.Server
	logger *slog.Logger
}

func (s *loggedDHCPServer) Demux(carrierData []byte, frameOffset int) error {
	if s.logger != nil {
		if frame, err := dhcpv4.NewFrame(carrierData[frameOffset:]); err == nil {
			var message dhcpv4.MessageType
			var clientIDLen int
			_ = frame.ForEachOption(func(_ int, option dhcpv4.OptNum, data []byte) error {
				switch option {
				case dhcpv4.OptMessageType:
					if len(data) == 1 {
						message = dhcpv4.MessageType(data[0])
					}
				case dhcpv4.OptClientIdentifier:
					clientIDLen = len(data)
				}
				return nil
			})
			s.logger.Info("dhcp received",
				slog.String("message", message.String()),
				slog.Uint64("xid", uint64(frame.XID())),
				slog.String("chaddr", ethernet.String(*frame.CHAddrAs6())),
				slog.Int("client_id_len", clientIDLen),
			)
		}
	}
	err := s.Server.Demux(carrierData, frameOffset)
	if err != nil && s.logger != nil {
		s.logger.Info("dhcp rejected", slog.String("err", err.Error()))
	}
	return err
}
