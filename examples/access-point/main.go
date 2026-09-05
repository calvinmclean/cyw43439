//go:build rp2040 || rp2350

package main

import (
	"errors"
	"log/slog"
	"machine"
	"time"

	"github.com/soypat/cyw43439"
	"github.com/soypat/lneto"
	"github.com/soypat/lneto/dhcp/dhcpv4"
	"github.com/soypat/lneto/ethernet"
	"github.com/soypat/lneto/x/xnet"
)

const (
	ssid     = "Pico W"
	password = "pico-password"
	channel  = 6
)

func main() {
	// Give the USB serial monitor time to connect after flashing.
	time.Sleep(2 * time.Second)
	logger := slog.New(slog.NewTextHandler(machine.Serial, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := cyw43439.DefaultWifiConfig()
	cfg.Logger = logger
	dev := cyw43439.NewPicoWDevice()
	if err := dev.Init(cfg); err != nil {
		panic("init failed: " + err.Error())
	}

	mac, err := dev.HardwareAddr6()
	if err != nil {
		panic("get hardware address: " + err.Error())
	}
	var stack xnet.StackAsync
	var dhcpServer dhcpv4.Server
	if err := configureNetwork(&stack, &dhcpServer, mac, logger); err != nil {
		panic("configure network: " + err.Error())
	}
	dev.RecvEthHandle(func(packet []byte) {
		if err := stack.IngressEthernet(packet); err != nil && !errors.Is(err, lneto.ErrPacketDrop) {
			logger.Error("network ingress failed", slog.String("err", err.Error()))
		}
	})

	if err := dev.StartAP(ssid, password, channel); err != nil {
		panic("start AP failed: " + err.Error())
	}

	logger.Info("access point started",
		slog.String("ssid", ssid),
		slog.Int("channel", channel),
		slog.String("addr", apAddr.String()),
	)
	var packetBuffer [ethernet.MaxFrameLength]byte
	for {
		packetReceived, err := dev.PollOne()
		if err != nil {
			logger.Error("poll failed", slog.String("err", err.Error()))
		}
		n, err := stack.EgressEthernet(packetBuffer[:])
		if err != nil {
			logger.Error("network egress failed", slog.String("err", err.Error()))
		} else if n > 0 {
			if err := dev.SendEth(packetBuffer[:n]); err != nil {
				logger.Error("send failed", slog.String("err", err.Error()))
			}
		}
		if !packetReceived && n == 0 {
			time.Sleep(time.Millisecond)
		}
	}
}
