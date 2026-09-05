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

// AccessPoint represents a Wi-Fi access point discovered during scanning.
// It intentionally matches the common fields returned by espradio.Scan.
type AccessPoint struct {
	SSID string
	RSSI int
}

// Scan performs an active scan for all visible Wi-Fi networks and returns the
// discovered access points. It blocks until the firmware reports completion or
// the default timeout expires.
func (d *Device) Scan() ([]AccessPoint, error) {
	err := d.acquire(modeWifi)
	defer d.release()
	if err != nil {
		return nil, err
	}

	var request [76]byte
	putScanRequest(&request)
	d.scanResults = nil
	d.scanDone = false
	d.scanErr = nil
	d.eventmask.Enable(whd.EvESCAN_RESULT)
	defer func() {
		d.eventmask.Disable(whd.EvESCAN_RESULT)
		d.scanResults = nil
	}()

	if err := d.set_iovar_n("escan", whd.IF_STA, request[:]); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(15 * time.Second)
	for !d.scanDone {
		if time.Until(deadline) <= 0 {
			return nil, ErrScanTimeout
		}
		if err := d.check_status(d._rxBuf[:]); err != nil {
			return nil, err
		}
		if !d.scanDone {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if d.scanErr != nil {
		return nil, d.scanErr
	}
	return d.scanResults, nil
}

func putScanRequest(request *[76]byte) {
	dst := request[:]
	binary.LittleEndian.PutUint32(dst[0:4], 1) // ESCAN_REQ_VERSION
	binary.LittleEndian.PutUint16(dst[4:6], 1) // WL_SCAN_ACTION_START
	binary.LittleEndian.PutUint16(dst[6:8], 1) // sync ID
	for i := 44; i < 50; i++ {
		dst[i] = 0xff // Any BSSID.
	}
	dst[50] = 2 // WICED_BSS_TYPE_ANY
	for _, offset := range [...]int{52, 56, 60, 64} {
		binary.LittleEndian.PutUint32(dst[offset:offset+4], ^uint32(0))
	}
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
	d.scanResults = append(d.scanResults, AccessPoint{
		SSID: result.SSIDString(),
		RSSI: int(result.RSSI),
	})
	return nil
}
