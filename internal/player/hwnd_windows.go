//go:build windows
// +build windows

package player

import (
	"syscall"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procGetClientRect            = user32.NewProc("GetClientRect")
	procMoveWindow               = user32.NewProc("MoveWindow")
	procShowWindow               = user32.NewProc("ShowWindow")
)

const (
	WS_CHILD        = 0x40000000
	WS_VISIBLE      = 0x10000000
	WS_CLIPCHILDREN = 0x02000000
	SW_SHOW         = 5
)

// GetMainWindowHWND obtiene el HWND de la ventana principal de Wails
// buscando una ventana que pertenezca al proceso actual
func GetMainWindowHWND() (uintptr, error) {
	var hwnd uintptr
	currentPID := uint32(syscall.Getpid())

	// Callback para EnumWindows
	cb := syscall.NewCallback(func(h uintptr, p uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessId.Call(h, uintptr(unsafe.Pointer(&pid)))

		if pid == currentPID {
			// Obtener título de la ventana
			buf := make([]uint16, 256)
			procGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
			title := syscall.UTF16ToString(buf)

			// Si tiene título (ventana principal, no popup)
			if len(title) > 0 {
				hwnd = h
				return 0 // Detener enumeración
			}
		}
		return 1 // Continuar enumeración
	})

	procEnumWindows.Call(cb, 0)
	return hwnd, nil
}

// RECT estructura para GetClientRect
type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// CreateMPVChildWindow crea una ventana hija dentro de la ventana principal de Wails
// específicamente para renderizar mpv
func CreateMPVChildWindow() (uintptr, error) {
	// Obtener ventana principal de Wails
	parentHWND, err := GetMainWindowHWND()
	if err != nil || parentHWND == 0 {
		return 0, err
	}

	// Obtener dimensiones del cliente de la ventana padre
	var rect RECT
	procGetClientRect.Call(parentHWND, uintptr(unsafe.Pointer(&rect)))

	width := rect.Right - rect.Left
	height := rect.Bottom - rect.Top

	// Crear child window
	className, _ := syscall.UTF16PtrFromString("STATIC")
	childHWND, _, _ := procCreateWindowExW.Call(
		0,                             // dwExStyle
		uintptr(unsafe.Pointer(className)), // lpClassName
		0,                             // lpWindowName
		WS_CHILD|WS_VISIBLE|WS_CLIPCHILDREN, // dwStyle
		0,                             // x
		0,                             // y
		uintptr(width),                // width
		uintptr(height),               // height
		parentHWND,                    // hWndParent
		0,                             // hMenu
		0,                             // hInstance
		0,                             // lpParam
	)

	if childHWND == 0 {
		return 0, syscall.GetLastError()
	}

	// Mostrar ventana
	procShowWindow.Call(childHWND, SW_SHOW)

	return childHWND, nil
}

// DestroyMPVChildWindow destruye la ventana hija creada para mpv
func DestroyMPVChildWindow(hwnd uintptr) error {
	if hwnd == 0 {
		return nil
	}

	ret, _, err := procDestroyWindow.Call(hwnd)
	if ret == 0 {
		return err
	}
	return nil
}
