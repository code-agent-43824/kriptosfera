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
	wsOverlapped       = 0x00000000
	wsCaption          = 0x00C00000
	wsVisible          = 0x10000000
	wsChild            = 0x40000000
	ssLeft             = 0x00000000
	ssLeftNowordwrap   = 0x0000000c
	wmDestroy          = 0x0002
	wmClose            = 0x0010
	wmPaint            = 0x000f
	wmCtlColorStatic   = 0x0138
	wmProgressClose    = 0x8001
	swShow             = 5
	cwUseDefault       = 0x80000000
	cwUseDefaultCoord  = -2147483648
	colorWindowFrame   = 6
	colorSteamBg       = 0x1b2838
	colorSteamPanel    = 0x213347
	colorSteamFill     = 0x66c0f4
	colorSteamTrack    = 0x2a475e
	colorTextPrimary   = 0x00f2f6fb
	colorTextSecondary = 0x00b8c6d0
	transparentBkMode  = 1
	windowClassName    = "KriptosferaProgressWindow"
	windowTitle        = "Kriptosfera"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	gdi32                   = syscall.NewLazyDLL("gdi32.dll")
	procBeginPaint          = user32.NewProc("BeginPaint")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procEndPaint            = user32.NewProc("EndPaint")
	procFillRect            = user32.NewProc("FillRect")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procGetStockObject      = gdi32.NewProc("GetStockObject")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procInvalidateRect      = user32.NewProc("InvalidateRect")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procSetBkMode           = gdi32.NewProc("SetBkMode")
	procSetTextColor        = gdi32.NewProc("SetTextColor")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	progressWindowClassOnce sync.Once
	progressWndProc         = syscall.NewCallback(progressWindowProc)
	progressWindowStates    sync.Map
)

type point struct {
	X int32
	Y int32
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type paintStruct struct {
	Hdc         uintptr
	Erase       int32
	Paint       rect
	Restore     int32
	IncUpdate   int32
	Reserved    [32]byte
}

type msg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
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
	logger      *logging.Logger
	readyOnce   sync.Once
	ready       chan struct{}
	closed      chan struct{}
	hwnd        uintptr
	titleLabel  uintptr
	statusLabel uintptr
	detailLabel uintptr
	mu          sync.Mutex
	dead        bool
	statusText  string
	detailText  string
	done        int64
	total       int64
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
	r.mu.Lock()
	if r.dead {
		r.mu.Unlock()
		return
	}
	r.statusText = text
	if r.total > 0 && r.done >= r.total {
		r.done = 0
		r.total = 0
		r.detailText = ""
	}
	statusLabel := r.statusLabel
	hwnd := r.hwnd
	r.mu.Unlock()
	if statusLabel != 0 {
		procSetWindowTextW.Call(statusLabel, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
	}
	invalidateProgressWindow(hwnd)
}

func (r *windowsProgressReporter) SetDownloadProgress(done, total int64) {
	r.ensureWindow("Подготовка загрузки компонентов...")
	detail := formatDownloadProgress(done, total)

	r.mu.Lock()
	if r.dead {
		r.mu.Unlock()
		return
	}
	r.statusText = "Загрузка компонентов"
	r.detailText = detail
	r.done = done
	r.total = total
	statusLabel := r.statusLabel
	detailLabel := r.detailLabel
	hwnd := r.hwnd
	r.mu.Unlock()

	if statusLabel != 0 {
		procSetWindowTextW.Call(statusLabel, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Загрузка компонентов"))))
	}
	if detailLabel != 0 {
		procSetWindowTextW.Call(detailLabel, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(detail))))
	}
	invalidateProgressWindow(hwnd)
}

func (r *windowsProgressReporter) Close() error {
	r.mu.Lock()
	dead := r.dead
	hwnd := r.hwnd
	closed := r.closed
	r.mu.Unlock()
	if dead || hwnd == 0 {
		return nil
	}
	procPostMessageW.Call(hwnd, wmProgressClose, 0, 0)
	<-closed
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

	width := int32(520)
	height := int32(190)
	x := centerWindowCoordinate(width, 0)
	y := centerWindowCoordinate(height, 1)

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsOverlapped|wsCaption|wsVisible),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		0,
		0,
		instance,
		0,
	)
	if hwnd == 0 {
		close(r.ready)
		return
	}

	titleText := syscall.StringToUTF16Ptr("Подготовка компонентов")
	statusText := syscall.StringToUTF16Ptr(initialText)
	detailText := syscall.StringToUTF16Ptr("Это нужно только для первой загрузки или после обновления payload")
	staticClass := syscall.StringToUTF16Ptr("STATIC")

	titleLabel, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(titleText)),
		uintptr(wsChild|wsVisible|ssLeft),
		24,
		22,
		360,
		24,
		hwnd,
		0,
		instance,
		0,
	)
	statusLabel, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(statusText)),
		uintptr(wsChild|wsVisible|ssLeft),
		24,
		58,
		460,
		24,
		hwnd,
		0,
		instance,
		0,
	)
	detailLabel, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(detailText)),
		uintptr(wsChild|wsVisible|ssLeft|ssLeftNowordwrap),
		24,
		132,
		460,
		20,
		hwnd,
		0,
		instance,
		0,
	)

	r.mu.Lock()
	r.hwnd = hwnd
	r.titleLabel = titleLabel
	r.statusLabel = statusLabel
	r.detailLabel = detailLabel
	r.statusText = initialText
	progressWindowStates.Store(hwnd, r)
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

	progressWindowStates.Delete(hwnd)
	r.mu.Lock()
	r.dead = true
	r.hwnd = 0
	r.titleLabel = 0
	r.statusLabel = 0
	r.detailLabel = 0
	r.mu.Unlock()
	if r.logger != nil {
		r.logger.Info("progress window closed")
	}
}

func (r *windowsProgressReporter) snapshot() (int64, int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.done, r.total
}

func centerWindowCoordinate(size int32, axis int) int32 {
	metric := uintptr(0)
	if axis == 0 {
		metric = 0
	} else {
		metric = 1
	}
	value, _, _ := procGetSystemMetrics.Call(metric)
	if value == 0 {
		return cwUseDefaultCoord
	}
	centered := int32(int(value)-int(size)) / 2
	if centered < 0 {
		return cwUseDefaultCoord
	}
	return centered
}

func invalidateProgressWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	procInvalidateRect.Call(hwnd, 0, 1)
	procUpdateWindow.Call(hwnd)
}

func registerProgressWindowClass() {
	progressWindowClassOnce.Do(func() {
		instance, _, _ := procGetModuleHandleW.Call(0)
		className := syscall.StringToUTF16Ptr(windowClassName)
		wc := wndClassEx{
			Size:       uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:    progressWndProc,
			Instance:   instance,
			Background: uintptr(colorWindowFrame + 1),
			ClassName:  className,
		}
		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})
}

func progressWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmProgressClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	case wmCtlColorStatic:
		return handleStaticColors(hwnd, wParam, lParam)
	case wmPaint:
		drawProgressWindow(hwnd)
		return 0
	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}
}

func handleStaticColors(hwnd, wParam, lParam uintptr) uintptr {
	value, ok := progressWindowStates.Load(hwnd)
	if !ok {
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(wmCtlColorStatic), wParam, lParam)
		return ret
	}
	reporter := value.(*windowsProgressReporter)
	hdc := wParam
	control := lParam

	procSetBkMode.Call(hdc, transparentBkMode)
	switch control {
	case reporter.titleLabel:
		procSetTextColor.Call(hdc, colorTextPrimary)
	case reporter.statusLabel:
		procSetTextColor.Call(hdc, colorTextPrimary)
	default:
		procSetTextColor.Call(hdc, colorTextSecondary)
	}
	brush, _, _ := procGetStockObject.Call(5)
	return brush
}

func drawProgressWindow(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	fillRectColor(hdc, rect{Left: 0, Top: 0, Right: 520, Bottom: 190}, colorSteamBg)
	fillRectColor(hdc, rect{Left: 24, Top: 92, Right: 496, Bottom: 116}, colorSteamTrack)

	value, ok := progressWindowStates.Load(hwnd)
	if !ok {
		return
	}
	reporter := value.(*windowsProgressReporter)
	done, total := reporter.snapshot()
	if total > 0 && done > 0 {
		if done > total {
			done = total
		}
		barWidth := int32(float64(472) * (float64(done) / float64(total)))
		if barWidth < 8 {
			barWidth = 8
		}
		fillRectColor(hdc, rect{Left: 24, Top: 92, Right: 24 + barWidth, Bottom: 116}, colorSteamFill)
	}
	fillRectColor(hdc, rect{Left: 24, Top: 120, Right: 496, Bottom: 121}, colorSteamPanel)
}

func fillRectColor(hdc uintptr, area rect, color uintptr) {
	brush, _, _ := procCreateSolidBrush.Call(color)
	if brush == 0 {
		return
	}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&area)), brush)
	procDeleteObject.Call(brush)
}
