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
	count := 0
	err := dev.Scan(cyw43439.ScanOptions{}, func(result cyw43439.ScanResult) {
		count++
		println(count, result.SSIDString(), "channel", result.Channel, "RSSI", result.RSSI, "auth", result.AuthMode)
	})
	if err != nil {
		panic("Wi-Fi scan: " + err.Error())
	}
	println("scan complete:", count, "networks")

	for {
		time.Sleep(time.Hour)
	}
}
