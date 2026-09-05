//go:build tinygo && rp2040

package main

import (
	"time"

	"github.com/soypat/cyw43439"
)

func main() {
	// Give the USB serial connection time to enumerate before printing output.
	time.Sleep(time.Second)

	dev := cyw43439.NewPicoWDevice()
	if err := dev.Init(cyw43439.DefaultWifiConfig()); err != nil {
		panic("Wi-Fi init: " + err.Error())
	}

	println("scanning Wi-Fi networks...")
	accessPoints, err := dev.Scan()
	if err != nil {
		panic("Wi-Fi scan: " + err.Error())
	}
	for i, ap := range accessPoints {
		println(i+1, ap.SSID, "RSSI", ap.RSSI)
	}
	println("scan complete:", len(accessPoints), "networks")

	for {
		time.Sleep(time.Hour)
	}
}
