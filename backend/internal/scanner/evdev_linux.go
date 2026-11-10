//go:build linux

package scanner

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"sync"
	"syscall"
	"time"
)

// Linux 64-bit struct input_event:
// timeval (16 bytes: 8 sec, 8 usec) + uint16 type (2) + uint16 code (2) + int32 value (4) = 24 bytes
type inputEvent struct {
	Sec   int64
	Usec  int64
	Type  uint16
	Code  uint16
	Value int32
}

const (
	evKey         = 0x01
	keyBackspace  = 14
	keyEnter      = 28
	keyLeftShift  = 42
	keyRightShift = 54
	keyKpEnter    = 96
)

var scancodeMap = map[uint16][2]rune{
	2:  {'1', '!'},
	3:  {'2', '@'},
	4:  {'3', '#'},
	5:  {'4', '$'},
	6:  {'5', '%'},
	7:  {'6', '^'},
	8:  {'7', '&'},
	9:  {'8', '*'},
	10: {'9', '('},
	11: {'0', ')'},
	12: {'-', '_'},
	13: {'=', '+'},
	16: {'q', 'Q'},
	17: {'w', 'W'},
	18: {'e', 'E'},
	19: {'r', 'R'},
	20: {'t', 'T'},
	21: {'y', 'Y'},
	22: {'u', 'U'},
	23: {'i', 'I'},
	24: {'o', 'O'},
	25: {'p', 'P'},
	26: {'[', '{'},
	27: {']', '}'},
	30: {'a', 'A'},
	31: {'s', 'S'},
	32: {'d', 'D'},
	33: {'f', 'F'},
	34: {'g', 'G'},
	35: {'h', 'H'},
	36: {'j', 'J'},
	37: {'k', 'K'},
	38: {'l', 'L'},
	39: {';', ':'},
	40: {'\'', '"'},
	41: {'`', '~'},
	43: {'\\', '|'},
	44: {'z', 'Z'},
	45: {'x', 'X'},
	46: {'c', 'C'},
	47: {'v', 'V'},
	48: {'b', 'B'},
	49: {'n', 'N'},
	50: {'m', 'M'},
	51: {',', '<'},
	52: {'.', '>'},
	53: {'/', '?'},
	57: {' ', ' '},
	// Keypad
	82: {'0', '0'},
	79: {'1', '1'},
	80: {'2', '2'},
	81: {'3', '3'},
	75: {'4', '4'},
	76: {'5', '5'},
	77: {'6', '6'},
	71: {'7', '7'},
	72: {'8', '8'},
	73: {'9', '9'},
	83: {'.', '.'},
	98: {'/', '/'},
	55: {'*', '*'},
	74: {'-', '-'},
	78: {'+', '+'},
}

// EvdevListener escucha eventos de teclado crudos a nivel de kernel desde /dev/input.
type EvdevListener struct {
	mu         sync.Mutex
	running    bool
	stopCh     chan struct{}
	callback   func(code string, durationMs float64, source string)
	devicePath string
}

// NewEvdevListener crea una nueva instancia del listener evdev.
func NewEvdevListener(cb func(code string, durationMs float64, source string)) *EvdevListener {
	return &EvdevListener{
		callback: cb,
	}
}

// Start inicia el bucle asíncrono en background.
func (l *EvdevListener) Start() bool {
	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return true
	}
	l.stopCh = make(chan struct{})
	l.running = true
	l.mu.Unlock()

	go l.supervisorLoop()
	return true
}

// Stop detiene el monitor.
func (l *EvdevListener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.running {
		return
	}
	l.running = false
	close(l.stopCh)
}

// IsRunning indica si está activo.
func (l *EvdevListener) IsRunning() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.running
}

func (l *EvdevListener) supervisorLoop() {
	for {
		l.mu.Lock()
		if !l.running {
			l.mu.Unlock()
			return
		}
		l.mu.Unlock()

		_, devNode := FindSymbolScannerDevice()
		if devNode == "" {
			time.Sleep(3 * time.Second)
			continue
		}

		f, err := os.OpenFile(devNode, os.O_RDONLY, 0)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		l.mu.Lock()
		l.devicePath = devNode
		l.mu.Unlock()
		log.Printf("[evdev] Conectado a escáner USB en %s", devNode)

		// Leer eventos hasta error o cierre
		l.readEvents(f)
		f.Close()

		l.mu.Lock()
		l.devicePath = ""
		l.mu.Unlock()

		select {
		case <-l.stopCh:
			return
		case <-time.After(1 * time.Second):
		}
	}
}

func (l *EvdevListener) readEvents(f *os.File) {
	buf := make([]byte, 24)
	var shiftPressed bool
	var currentRunes []rune
	var startTime time.Time

	for {
		select {
		case <-l.stopCh:
			return
		default:
		}

		// Establecer timeout corto para chequear stopCh
		_ = f.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, err := io.ReadFull(f, buf)
		if err != nil {
			if netErr, ok := err.(*os.PathError); ok && netErr.Timeout() {
				continue
			}
			if err == io.EOF || err == syscall.EBADF {
				return
			}
			return
		}

		ev := inputEvent{
			Sec:   int64(binary.LittleEndian.Uint64(buf[0:8])),
			Usec:  int64(binary.LittleEndian.Uint64(buf[8:16])),
			Type:  binary.LittleEndian.Uint16(buf[16:18]),
			Code:  binary.LittleEndian.Uint16(buf[18:20]),
			Value: int32(binary.LittleEndian.Uint32(buf[20:24])),
		}

		if ev.Type != evKey {
			continue
		}

		if ev.Code == keyLeftShift || ev.Code == keyRightShift {
			shiftPressed = ev.Value > 0
			continue
		}

		// Solo procesar key press (1) o repeat (2)
		if ev.Value == 0 {
			continue
		}

		now := time.Now()
		if len(currentRunes) == 0 {
			startTime = now
		}

		if ev.Code == keyEnter || ev.Code == keyKpEnter {
			if len(currentRunes) > 0 {
				durationMs := float64(now.Sub(startTime).Microseconds()) / 1000.0
				scanned := string(currentRunes)
				log.Printf("[evdev] Código recibido de hardware: '%s' en %.1fms", scanned, durationMs)

				if l.callback != nil {
					l.callback(scanned, durationMs, "evdev_hardware")
				}

				currentRunes = currentRunes[:0]
			}
			continue
		}

		if ev.Code == keyBackspace {
			if len(currentRunes) > 0 {
				currentRunes = currentRunes[:len(currentRunes)-1]
			}
			continue
		}

		if pair, ok := scancodeMap[ev.Code]; ok {
			ch := pair[0]
			if shiftPressed {
				ch = pair[1]
			}
			currentRunes = append(currentRunes, ch)
		}
	}
}
