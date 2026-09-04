package cyw43439

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/soypat/cyw43439/whd"
)

var (
	ErrScanTimeout = errors.New("wifi scan timed out")
	ErrScanFailed  = errors.New("wifi scan failed")
)

// ScanOptions configures a Wi-Fi network scan. Its zero value performs an
// active scan for all visible networks using firmware-default timings.
type ScanOptions struct {
	// SSID restricts results to a network name. An empty SSID scans all names.
	SSID string
	// BSSID restricts results to an access point. A zero address scans all APs.
	BSSID [6]byte
	// Passive listens for beacons instead of transmitting probe requests.
	Passive bool
	// Probes is the number of probes sent per channel during an active scan.
	// Zero uses the firmware default.
	Probes uint16
	// DwellTime is the time spent on each channel. Zero uses the firmware default.
	DwellTime time.Duration
	// HomeTime is the time spent on the home channel. Zero uses the firmware default.
	HomeTime time.Duration
	// Timeout bounds the complete scan. Zero defaults to 15 seconds.
	Timeout time.Duration
}

// ScanResult describes one access point found during a scan. AuthMode uses
// bit 0 for WEP, bit 1 for WPA, and bit 2 for WPA2; zero indicates an open AP.
type ScanResult = whd.EventScanResult

// Scan result authentication flags. AuthMode can contain more than one flag.
const (
	ScanAuthOpen uint8 = 0
	ScanAuthWEP  uint8 = 1 << 0
	ScanAuthWPA  uint8 = 1 << 1
	ScanAuthWPA2 uint8 = 1 << 2
)

// Scan performs a Wi-Fi network scan and calls found once for each result.
// Scan blocks until the firmware reports completion or the timeout expires.
// The callback runs while the device is locked and must not call Device methods.
// A nil callback is allowed.
func (d *Device) Scan(options ScanOptions, found func(ScanResult)) error {
	err := d.acquire(modeWifi)
	defer d.release()
	if err != nil {
		return err
	}

	var request [76]byte
	if err := putScanOptions(request[:], options); err != nil {
		return err
	}
	d.scanCallback = found
	d.scanDone = false
	d.scanErr = nil
	d.eventmask.Enable(whd.EvESCAN_RESULT)
	defer func() {
		d.eventmask.Disable(whd.EvESCAN_RESULT)
		d.scanCallback = nil
	}()

	if err := d.set_iovar_n("escan", whd.IF_STA, request[:]); err != nil {
		return err
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for !d.scanDone {
		if time.Until(deadline) <= 0 {
			return ErrScanTimeout
		}
		if err := d.check_status(d._rxBuf[:]); err != nil {
			return err
		}
		if !d.scanDone {
			time.Sleep(10 * time.Millisecond)
		}
	}
	return d.scanErr
}

func putScanOptions(dst []byte, options ScanOptions) error {
	if len(dst) < 76 {
		return errors.New("scan options buffer too small")
	}
	if len(options.SSID) > 32 {
		return errors.New("scan SSID too long")
	}
	if options.DwellTime < 0 || options.HomeTime < 0 || options.Timeout < 0 {
		return errors.New("negative scan duration")
	}
	dwell, err := scanMilliseconds(options.DwellTime)
	if err != nil {
		return err
	}
	home, err := scanMilliseconds(options.HomeTime)
	if err != nil {
		return err
	}
	for i := range dst[:76] {
		dst[i] = 0
	}
	binary.LittleEndian.PutUint32(dst[0:4], 1) // ESCAN_REQ_VERSION
	binary.LittleEndian.PutUint16(dst[4:6], 1) // WL_SCAN_ACTION_START
	binary.LittleEndian.PutUint16(dst[6:8], 1) // sync ID
	binary.LittleEndian.PutUint32(dst[8:12], uint32(len(options.SSID)))
	copy(dst[12:44], options.SSID)
	bssid := options.BSSID
	if bssid == [6]byte{} {
		bssid = [6]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	}
	copy(dst[44:50], bssid[:])
	dst[50] = 2 // WICED_BSS_TYPE_ANY
	if options.Passive {
		dst[51] = 1
	}
	nprobes := int32(-1)
	if options.Probes != 0 {
		nprobes = int32(options.Probes)
	}
	binary.LittleEndian.PutUint32(dst[52:56], uint32(nprobes))
	activeTime, passiveTime := int32(-1), int32(-1)
	if options.Passive {
		passiveTime = dwell
	} else {
		activeTime = dwell
	}
	binary.LittleEndian.PutUint32(dst[56:60], uint32(activeTime))
	binary.LittleEndian.PutUint32(dst[60:64], uint32(passiveTime))
	binary.LittleEndian.PutUint32(dst[64:68], uint32(home))
	return nil
}

func scanMilliseconds(duration time.Duration) (int32, error) {
	if duration == 0 {
		return -1, nil
	}
	milliseconds := duration / time.Millisecond
	if milliseconds > 1<<31-1 {
		return 0, errors.New("scan duration too long")
	}
	return int32(milliseconds), nil
}

func (d *Device) handleScanEvent(status whd.EStatus, payload []byte) error {
	if status != whd.EStatusPartial {
		d.scanDone = true
		if status != whd.EStatusSuccess {
			d.scanErr = ErrScanFailed
		}
		return nil
	}
	result, err := whd.ParseScanResult(_busOrder, payload)
	if err != nil {
		return err
	}
	if d.scanCallback != nil {
		d.scanCallback(result)
	}
	return nil
}
