package whd

import (
	"encoding/binary"
	"testing"
)

func TestParseAsyncEvent(t *testing.T) {
	var buf [48]byte
	for i := range buf {
		buf[i] = byte(i)
	}
	ev, err := ParseAsyncEvent(binary.BigEndian, buf[:])
	if err != nil {
		t.Fatal(err)
	}
	if ev.Flags != 515 {
		t.Error("bad flags")
	}
	if ev.EventType != 67438087 {
		t.Error("bad event type")
	}
	if ev.Status != 134810123 {
		t.Error("bad status")
	}
	if ev.Reason != 202182159 {
		t.Error("bad reason")
	}
}

func TestParseScanResult(t *testing.T) {
	// Use an intentionally unaligned slice. Scan event payloads are only
	// guaranteed to be aligned to two bytes on RP2040.
	storage := make([]byte, 1+12+136)
	buf := storage[1:]
	const bss = 12
	binary.LittleEndian.PutUint32(buf[bss+4:], 136)
	copy(buf[bss+8:], []byte{1, 2, 3, 4, 5, 6})
	binary.LittleEndian.PutUint16(buf[bss+16:], DOT11_CAP_PRIVACY)
	buf[bss+18] = 7
	copy(buf[bss+19:], "testnet")
	binary.LittleEndian.PutUint16(buf[bss+72:], 0x100b)
	binary.LittleEndian.PutUint16(buf[bss+78:], 0xffd6)
	binary.LittleEndian.PutUint16(buf[bss+116:], 128)
	ies := []byte{DOT11_IE_ID_RSN, 0, DOT11_IE_ID_VENDOR_SPECIFIC, 4, 0, 0x50, 0xf2, 1}
	binary.LittleEndian.PutUint32(buf[bss+120:], uint32(len(ies)))
	copy(buf[bss+128:], ies)

	result, err := ParseScanResult(binary.BigEndian, buf)
	if err != nil {
		t.Fatal(err)
	}
	if result.SSIDString() != "testnet" {
		t.Fatalf("SSID: got %q", result.SSIDString())
	}
	if result.BSSID != [6]byte{1, 2, 3, 4, 5, 6} {
		t.Fatalf("BSSID: got %v", result.BSSID)
	}
	if result.Channel != 11 || result.RSSI != -42 || result.AuthMode != 7 {
		t.Fatalf("metadata: got channel=%d RSSI=%d auth=%d", result.Channel, result.RSSI, result.AuthMode)
	}
}

func TestParseScanResultRejectsInvalidLengths(t *testing.T) {
	buf := make([]byte, 12+128)
	const bss = 12
	binary.LittleEndian.PutUint32(buf[bss+4:], 128)
	buf[bss+18] = 33
	if _, err := ParseScanResult(binary.LittleEndian, buf); err == nil {
		t.Fatal("expected invalid SSID length error")
	}

	buf[bss+18] = 0
	binary.LittleEndian.PutUint16(buf[bss+116:], 127)
	binary.LittleEndian.PutUint32(buf[bss+120:], 2)
	if _, err := ParseScanResult(binary.LittleEndian, buf); err == nil {
		t.Fatal("expected invalid IE bounds error")
	}
}
