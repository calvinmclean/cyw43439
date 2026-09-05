package cyw43439

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/soypat/cyw43439/whd"
)

func TestPutScanRequest(t *testing.T) {
	var got [76]byte
	putScanRequest(&got)
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
