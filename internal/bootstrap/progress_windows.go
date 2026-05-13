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
	wsTabStop          = 0x00010000
	ssLeft             = 0x00000000
	ssLeftNowordwrap   = 0x0000000c
	wmDestroy          = 0x0002
	wmClose            = 0x0010
	wmSetFont          = 0x0030
	wmCtlColorStatic   = 0x0138
	wmProgressClose    = 0x8001
	pbmSetRange32      = 0x0406
	pbmSetPos          = 0x0402
	swShow             = 5
	cwUseDefault       = 0x80000000
	cwUseDefaultCoord  = -2147483648
	fwNormal           = 400
	fwSemiBold         = 600
	logPixelsY         = 90
	defaultCharset     = 1
	outDefaultPrecis   = 0
	clipDefaultPrecis  = 0
	clearTypeQuality   = 5
	defaultPitchFamily = 0
	dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3)
	colorBtnFace       = 15
	colorWindowText    = 8
	colorGrayText      = 17
	transparentBkMode  = 1
	windowClassName    = "KriptosferaProgressWindow"
	windowTitle        = "Kriptosfera"
	progressClassName  = "msctls_progress32"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	gdi32                   = syscall.NewLazyDLL("gdi32.dll")
	comctl32                = syscall.NewLazyDLL("comctl32.dll")
	procBeginPaint          = user32.NewProc("BeginPaint")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procGetDC               = user32.NewProc("GetDC")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procGetSysColorBrush    = user32.NewProc("GetSysColorBrush")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procGetSysColor         = user32.NewProc("GetSysColor")
	procInitCommonControls  = comctl32.NewProc("InitCommonControls")
	procMulDiv              = kernel32.NewProc("MulDiv")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procSendMessageW        = user32.NewProc("SendMessageW")
	procSetProcessDPIAware  = user32.NewProc("SetProcessDPIAware")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetBkMode           = gdi32.NewProc("SetBkMode")
	procSetTextColor        = gdi32.NewProc("SetTextColor")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procCreateFontW         = gdi32.NewProc("CreateFontW")
	procGetDeviceCaps       = gdi32.NewProc("GetDeviceCaps")
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
	progressBar uintptr
	titleFont   uintptr
	bodyFont    uintptr
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
	progressBar := r.progressBar
	r.mu.Unlock()
	if statusLabel != 0 {
		procSetWindowTextW.Call(statusLabel, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
	}
	if progressBar != 0 {
		procSendMessageW.Call(progressBar, pbmSetPos, 0, 0)
	}
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
	progressBar := r.progressBar
	r.mu.Unlock()

	if statusLabel != 0 {
		procSetWindowTextW.Call(statusLabel, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Загрузка компонентов"))))
	}
	if detailLabel != 0 {
		procSetWindowTextW.Call(detailLabel, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(detail))))
	}
	if progressBar != 0 {
		procSendMessageW.Call(progressBar, pbmSetRange32, 0, uintptr(maxInt64(total, 1)))
		procSendMessageW.Call(progressBar, pbmSetPos, uintptr(done), 0)
	}
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
	enableDPIAwareness()
	procInitCommonControls.Call()

	registerProgressWindowClass()
	instance, _, _ := procGetModuleHandleW.Call(0)
	className := syscall.StringToUTF16Ptr(windowClassName)
	title := syscall.StringToUTF16Ptr(windowTitle)
	dpi := currentScreenDPI()

	width := scaleInt32(560, dpi)
	height := scaleInt32(220, dpi)
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
	progressClass := syscall.StringToUTF16Ptr(progressClassName)
	titleFont := createSegoeUIFont(hwnd, 12, fwSemiBold)
	bodyFont := createSegoeUIFont(hwnd, 10, fwNormal)

	titleLabel, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(titleText)),
		uintptr(wsChild|wsVisible|ssLeft),
		uintptr(scaleInt32(28, dpi)),
		uintptr(scaleInt32(24, dpi)),
		uintptr(scaleInt32(420, dpi)),
		uintptr(scaleInt32(20, dpi)),
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
		uintptr(scaleInt32(28, dpi)),
		uintptr(scaleInt32(62, dpi)),
		uintptr(scaleInt32(500, dpi)),
		uintptr(scaleInt32(22, dpi)),
		hwnd,
		0,
		instance,
		0,
	)
	progressBar, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(progressClass)),
		0,
		uintptr(wsChild|wsVisible|wsTabStop),
		uintptr(scaleInt32(28, dpi)),
		uintptr(scaleInt32(98, dpi)),
		uintptr(scaleInt32(500, dpi)),
		uintptr(scaleInt32(22, dpi)),
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
		uintptr(scaleInt32(28, dpi)),
		uintptr(scaleInt32(142, dpi)),
		uintptr(scaleInt32(500, dpi)),
		uintptr(scaleInt32(24, dpi)),
		hwnd,
		0,
		instance,
		0,
	)

	applyControlFont(titleLabel, titleFont)
	applyControlFont(statusLabel, bodyFont)
	applyControlFont(detailLabel, bodyFont)

	r.mu.Lock()
	r.hwnd = hwnd
	r.titleLabel = titleLabel
	r.statusLabel = statusLabel
	r.detailLabel = detailLabel
	r.progressBar = progressBar
	r.titleFont = titleFont
	r.bodyFont = bodyFont
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
	r.progressBar = 0
	titleFont = r.titleFont
	bodyFont = r.bodyFont
	r.titleFont = 0
	r.bodyFont = 0
	r.mu.Unlock()
	if titleFont != 0 {
		procDeleteObject.Call(titleFont)
	}
	if bodyFont != 0 {
		procDeleteObject.Call(bodyFont)
	}
	if r.logger != nil {
		r.logger.Info("progress window closed")
	}
}

func enableDPIAwareness() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2)
		return
	}
	if procSetProcessDPIAware.Find() == nil {
		procSetProcessDPIAware.Call()
	}
}

func currentScreenDPI() int32 {
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return 96
	}
	defer procReleaseDC.Call(0, hdc)
	dpi, _, _ := procGetDeviceCaps.Call(hdc, logPixelsY)
	if dpi == 0 {
		return 96
	}
	return int32(dpi)
}

func scaleInt32(value int32, dpi int32) int32 {
	if dpi <= 0 || dpi == 96 {
		return value
	}
	scaled, _, _ := procMulDiv.Call(uintptr(value), uintptr(dpi), 96)
	if scaled == 0 {
		return value
	}
	return int32(scaled)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func mustSysColor(index uintptr) uintptr {
	value, _, _ := procGetSysColor.Call(index)
	return value
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

func registerProgressWindowClass() {
	progressWindowClassOnce.Do(func() {
		instance, _, _ := procGetModuleHandleW.Call(0)
		className := syscall.StringToUTF16Ptr(windowClassName)
		wc := wndClassEx{
			Size:       uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:    progressWndProc,
			Instance:   instance,
			Background: uintptr(colorBtnFace + 1),
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
		procSetTextColor.Call(hdc, mustSysColor(colorWindowText))
	case reporter.statusLabel:
		procSetTextColor.Call(hdc, mustSysColor(colorWindowText))
	default:
		procSetTextColor.Call(hdc, mustSysColor(colorGrayText))
	}
	brush, _, _ := procGetSysColorBrush.Call(colorBtnFace)
	return brush
}

func applyControlFont(hwnd, font uintptr) {
	if hwnd == 0 || font == 0 {
		return
	}
	procSendMessageW.Call(hwnd, wmSetFont, font, 1)
}

func createSegoeUIFont(hwnd uintptr, pointSize int32, weight uintptr) uintptr {
	hdc, _, _ := procGetDC.Call(hwnd)
	if hdc == 0 {
		return 0
	}
	defer procReleaseDC.Call(hwnd, hdc)
	deviceCaps := gdi32.NewProc("GetDeviceCaps")
	logPixelsY, _, _ := deviceCaps.Call(hdc, 90)
	height, _, _ := procMulDiv.Call(uintptr(pointSize), logPixelsY, 72)
	fontHeight := int32(height)
	if fontHeight > 0 {
		fontHeight = -fontHeight
	}
	font, _, _ := procCreateFontW.Call(
		uintptr(int32(fontHeight)),
		0,
		0,
		0,
		weight,
		0,
		0,
		0,
		defaultCharset,
		outDefaultPrecis,
		clipDefaultPrecis,
		clearTypeQuality,
		defaultPitchFamily,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Segoe UI"))),
	)
	return font
}
