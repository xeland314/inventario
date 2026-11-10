package scanner

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ScannerDeviceInfo contiene los metadatos técnicos del escáner Symbol USB.
type ScannerDeviceInfo struct {
	Connected         bool   `json:"connected"`
	VendorID          string `json:"vendor_id"`
	ProductID         string `json:"product_id"`
	VendorName        string `json:"vendor_name"`
	ProductName       string `json:"product_name"`
	SerialNumber      string `json:"serial_number"`
	DeviceSymlink     string `json:"device_symlink"`
	DeviceNode        string `json:"device_node"`
	IsReadable        bool   `json:"is_readable"`
	Driver            string `json:"driver"`
	BusType           string `json:"bus_type"`
	InterfaceClass    string `json:"interface_class"`
	InterfaceSubclass string `json:"interface_subclass"`
	InterfaceProtocol string `json:"interface_protocol"`
	OperationMode     string `json:"operation_mode"`
	Explanation       string `json:"explanation"`
	EvdevActive       bool   `json:"evdev_active"`
	WebWedgeActive    bool   `json:"web_wedge_active"`
}

// FindSymbolScannerDevice busca el nodo de dispositivo /dev/input/by-id o /dev/input/event*.
func FindSymbolScannerDevice() (symlink string, resolved string) {
	byIDDir := "/dev/input/by-id"
	if entries, err := os.ReadDir(byIDDir); err == nil {
		for _, e := range entries {
			nameLower := strings.ToLower(e.Name())
			if (strings.Contains(nameLower, "symbol") || strings.Contains(nameLower, "barcode") || strings.Contains(nameLower, "bar_code")) &&
				strings.Contains(nameLower, "event-kbd") {
				p := filepath.Join(byIDDir, e.Name())
				real, err := filepath.EvalSymlinks(p)
				if err == nil {
					return p, real
				}
				return p, p
			}
		}
	}

	// Búsqueda en /sys/class/input/event*/device/name
	matches, _ := filepath.Glob("/sys/class/input/event*/device/name")
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err == nil {
			nameLower := strings.ToLower(string(data))
			if strings.Contains(nameLower, "symbol") || strings.Contains(nameLower, "bar code") || strings.Contains(nameLower, "barcode") {
				// /sys/class/input/eventX/device/name -> eventX
				dir := filepath.Dir(filepath.Dir(m))
				eventName := filepath.Base(dir)
				devNode := filepath.Join("/dev/input", eventName)
				if _, err := os.Stat(devNode); err == nil {
					return devNode, devNode
				}
			}
		}
	}

	return "", ""
}

// GetScannerInfo inspecciona el estado del escáner Symbol USB en Linux.
func GetScannerInfo(evdevActive bool) ScannerDeviceInfo {
	symlink, devNode := FindSymbolScannerDevice()
	connected := devNode != ""

	info := ScannerDeviceInfo{
		Connected:         connected,
		VendorID:          "05e0",
		ProductID:         "1200",
		VendorName:        "Symbol Technologies, Inc.",
		ProductName:       "Symbol Bar Code Scanner",
		SerialNumber:      "N/A",
		DeviceSymlink:     symlink,
		DeviceNode:        devNode,
		IsReadable:        false,
		Driver:            "usbhid",
		BusType:           "usb",
		InterfaceClass:    "03 (Human Interface Device - HID)",
		InterfaceSubclass: "01 (Boot Interface)",
		InterfaceProtocol: "01 (Keyboard)",
		OperationMode:     "USB HID Keyboard Wedge (Emulación de Teclado)",
		Explanation: "El escáner Symbol opera como un teclado USB (Keyboard Wedge). " +
			"Decodifica códigos ópticos a ráfagas de pulsaciones de teclas en <50ms, " +
			"terminadas automáticamente con Enter. Funciona vía driver Linux evdev o receptor global web.",
		EvdevActive:    evdevActive,
		WebWedgeActive: true,
	}

	if !connected {
		return info
	}

	// Comprobar permisos de lectura
	if f, err := os.OpenFile(devNode, os.O_RDONLY, 0); err == nil {
		info.IsReadable = true
		f.Close()
	}

	// Obtener propiedades con udevadm
	if out, err := exec.Command("udevadm", "info", devNode).Output(); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "E: ") {
				parts := strings.SplitN(line[3:], "=", 2)
				if len(parts) == 2 {
					k, v := parts[0], parts[1]
					switch k {
					case "ID_MODEL":
						info.ProductName = strings.ReplaceAll(v, "_", " ")
					case "ID_SERIAL_SHORT":
						info.SerialNumber = v
					case "ID_USB_DRIVER":
						info.Driver = v
					case "ID_VENDOR_ID":
						info.VendorID = v
					case "ID_MODEL_ID":
						info.ProductID = v
					case "ID_BUS":
						info.BusType = v
					}
				}
			}
		}
	}

	return info
}
