package cyw43439

import (
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/soypat/cyw43439/whd"
)

func TestPutScanOptionsDefaults(t *testing.T) {
	var got [76]byte
	if err := putScanOptions(got[:], ScanOptions{}); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(got[0:]) != 1 || binary.LittleEndian.Uint16(got[4:]) != 1 || binary.LittleEndian.Uint16(got[6:]) != 1 {
		t.Fatal("invalid escan header")
	}
	if got[44] != 0xff || got[49] != 0xff {
		t.Fatalf("default BSSID is not broadcast: %v", got[44:50])
	}
	if got[50] != 2 || got[51] != 0 {
		t.Fatalf("invalid BSS/scan types: %v", got[50:52])
	}
	for _, offset := range []int{52, 56, 60, 64} {
		if binary.LittleEndian.Uint32(got[offset:]) != ^uint32(0) {
			t.Fatalf("offset %d does not use firmware default", offset)
		}
	}
}

func TestPutScanOptionsFiltersAndTiming(t *testing.T) {
	var got [76]byte
	opts := ScanOptions{
		SSID:      "network",
		BSSID:     [6]byte{1, 2, 3, 4, 5, 6},
		Passive:   true,
		Probes:    3,
		DwellTime: 40 * time.Millisecond,
		HomeTime:  25 * time.Millisecond,
	}
	if err := putScanOptions(got[:], opts); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(got[8:]) != 7 || string(got[12:19]) != "network" {
		t.Fatal("SSID filter not encoded")
	}
	if got[51] != 1 || binary.LittleEndian.Uint32(got[52:]) != 3 {
		t.Fatal("passive scan settings not encoded")
	}
	if binary.LittleEndian.Uint32(got[56:]) != ^uint32(0) || binary.LittleEndian.Uint32(got[60:]) != 40 || binary.LittleEndian.Uint32(got[64:]) != 25 {
		t.Fatal("scan timings not encoded")
	}
}

func TestPutScanOptionsValidation(t *testing.T) {
	var dst [76]byte
	if err := putScanOptions(dst[:], ScanOptions{SSID: "123456789012345678901234567890123"}); err == nil {
		t.Fatal("expected long SSID error")
	}
	if err := putScanOptions(dst[:], ScanOptions{DwellTime: -time.Millisecond}); err == nil {
		t.Fatal("expected negative duration error")
	}
}

func TestHandleScanCompletion(t *testing.T) {
	var d Device
	if err := d.handleScanEvent(whd.EStatusSuccess, nil); err != nil || !d.scanDone || d.scanErr != nil {
		t.Fatalf("successful completion: done=%t err=%v", d.scanDone, err)
	}
	d.scanDone = false
	if err := d.handleScanEvent(whd.EStatusAbort, nil); err != nil || !d.scanDone || !errors.Is(d.scanErr, ErrScanFailed) {
		t.Fatalf("failed completion: done=%t eventErr=%v scanErr=%v", d.scanDone, err, d.scanErr)
	}
}
