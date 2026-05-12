//go:build windows

package bootstrap

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

const (
	wsOverlapped      = 0x00000000
	wsCaption         = 0x00C00000
	wsSysMenu         = 0x00080000
	wsVisible         = 0x10000000
	wsChild           = 0x40000000
	ssLeft            = 0x00000000
	wmDestroy         = 0x0002
	wmClose         = 0x0010
	swShow          = 5
	cwUseDefault    = 0x80000000
	windowClassName = "KriptosferaProgressWindow"
	windowTitle     = "Kriptosfera"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	progressWindowClassOnce sync.Once
	progressWndProc         = syscall.NewCallback(progressWindowProc)
)

type point struct {
	X int32
	Y int32
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	LPrivate uint32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type windowsProgressReporter struct {
	logger    *logging.Logger
	readyOnce sync.Once
	ready     chan struct{}
	closed    chan struct{}
	hwnd      uintptr
	label     uintptr
	mu        sync.Mutex
	dead      bool
	lastText  string
}

func newPlatformProgressReporter(logger *logging.Logger) ProgressReporter {
	return &windowsProgressReporter{
		logger: logger,
		ready:  make(chan struct{}),
		closed: make(chan struct{}),
	}
}

func (r *windowsProgressReporter) SetStatus(text string) {
	if text == "" {
		return
	}
	r.ensureWindow(text)
	r.setText(text)
}

func (r *windowsProgressReporter) SetDownloadProgress(done, total int64) {
	r.SetStatus(formatDownloadProgress(done, total))
}

func (r *windowsProgressReporter) Close() error {
	r.mu.Lock()
	dead := r.dead
	hwnd := r.hwnd
	r.mu.Unlock()
	if dead || hwnd == 0 {
		return nil
	}
	procDestroyWindow.Call(hwnd)
	<-r.closed
	return nil
}

func (r *windowsProgressReporter) ensureWindow(initialText string) {
	r.readyOnce.Do(func() {
		go r.run(initialText)
		<-r.ready
	})
}

func (r *windowsProgressReporter) run(initialText string) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(r.closed)

	registerProgressWindowClass()
	instance, _, _ := procGetModuleHandleW.Call(0)
	className := syscall.StringToUTF16Ptr(windowClassName)
	title := syscall.StringToUTF16Ptr(windowTitle)

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsOverlapped|wsCaption|wsSysMenu|wsVisible),
		cwUseDefault,
		cwUseDefault,
		420,
		120,
		0,
		0,
		instance,
		0,
	)
	if hwnd == 0 {
		close(r.ready)
		return
	}

	labelText := syscall.StringToUTF16Ptr(initialText)
	staticClass := syscall.StringToUTF16Ptr("STATIC")
	label, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(labelText)),
		uintptr(wsChild|wsVisible|ssLeft),
		20,
		20,
		360,
		40,
		hwnd,
		0,
		instance,
		0,
	)

	r.mu.Lock()
	r.hwnd = hwnd
	r.label = label
	r.lastText = initialText
	r.mu.Unlock()

	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)
	close(r.ready)

	var message msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}

	r.mu.Lock()
	r.dead = true
	r.hwnd = 0
	r.label = 0
	r.mu.Unlock()
	if r.logger != nil {
		r.logger.Info("progress window closed")
	}
}

func (r *windowsProgressReporter) setText(text string) {
	r.mu.Lock()
	label := r.label
	dead := r.dead
	if !dead {
		r.lastText = text
	}
	r.mu.Unlock()
	if dead || label == 0 {
		return
	}
	procSetWindowTextW.Call(label, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
}

func registerProgressWindowClass() {
	progressWindowClassOnce.Do(func() {
		instance, _, _ := procGetModuleHandleW.Call(0)
		className := syscall.StringToUTF16Ptr(windowClassName)
		wc := wndClassEx{
			Size:      uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:   progressWndProc,
			Instance:  instance,
			ClassName: className,
		}
		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})
}

func progressWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}
}
