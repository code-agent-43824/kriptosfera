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
	wsBorder           = 0x00800000
	ssLeft             = 0x00000000
	ssLeftNowordwrap   = 0x0000000c
	wmDestroy          = 0x0002
	wmClose            = 0x0010
	wmPaint            = 0x000f
	wmSetFont          = 0x0030
	wmCtlColorStatic   = 0x0138
	wmProgressClose    = 0x8001
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
	colorWindowFrame   = 6
	colorSurface       = 0x00f7f8fa
	colorDivider       = 0x00e2e6ea
	colorProgressFill  = 0x00569fff
	colorProgressTrack = 0x00e7ebf0
	colorTextPrimary   = 0x00212929
	colorTextSecondary = 0x00616a73
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
	procGetDC               = user32.NewProc("GetDC")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procGetStockObject      = gdi32.NewProc("GetStockObject")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procInvalidateRect      = user32.NewProc("InvalidateRect")
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
	procCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
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
	enableDPIAwareness()

	registerProgressWindowClass()
	instance, _, _ := procGetModuleHandleW.Call(0)
	className := syscall.StringToUTF16Ptr(windowClassName)
	title := syscall.StringToUTF16Ptr(windowTitle)
	dpi := currentScreenDPI()

	width := scaleInt32(560, dpi)
	height := scaleInt32(176, dpi)
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
	titleFont := createSegoeUIFont(hwnd, 20, fwSemiBold)
	bodyFont := createSegoeUIFont(hwnd, 12, fwNormal)

	titleLabel, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(titleText)),
		uintptr(wsChild|wsVisible|ssLeft),
		uintptr(scaleInt32(28, dpi)),
		uintptr(scaleInt32(20, dpi)),
		uintptr(scaleInt32(420, dpi)),
		uintptr(scaleInt32(26, dpi)),
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
		uintptr(scaleInt32(58, dpi)),
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
		uintptr(scaleInt32(118, dpi)),
		uintptr(scaleInt32(500, dpi)),
		uintptr(scaleInt32(20, dpi)),
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

func (r *windowsProgressReporter) snapshot() (int64, int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.done, r.total
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
	brush, _, _ := procGetStockObject.Call(0)
	return brush
}

func drawProgressWindow(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	dpi := currentScreenDPI()
	windowWidth := scaleInt32(560, dpi)
	windowHeight := scaleInt32(176, dpi)
	barLeft := scaleInt32(28, dpi)
	barTop := scaleInt32(86, dpi)
	barWidth := scaleInt32(504, dpi)
	barHeight := scaleInt32(10, dpi)
	dividerTop := scaleInt32(110, dpi)

	fillRectColor(hdc, rect{Left: 0, Top: 0, Right: windowWidth, Bottom: windowHeight}, colorSurface)
	fillRectColor(hdc, rect{Left: barLeft, Top: barTop, Right: barLeft + barWidth, Bottom: barTop + barHeight}, colorProgressTrack)

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
		fillWidth := int32(float64(barWidth) * (float64(done) / float64(total)))
		if fillWidth < scaleInt32(4, dpi) {
			fillWidth = scaleInt32(4, dpi)
		}
		fillRectColor(hdc, rect{Left: barLeft, Top: barTop, Right: barLeft + fillWidth, Bottom: barTop + barHeight}, colorProgressFill)
	}
	fillRectColor(hdc, rect{Left: barLeft, Top: dividerTop, Right: barLeft + barWidth, Bottom: dividerTop + 1}, colorDivider)
}

func fillRectColor(hdc uintptr, area rect, color uintptr) {
	brush, _, _ := procCreateSolidBrush.Call(color)
	if brush == 0 {
		return
	}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&area)), brush)
	procDeleteObject.Call(brush)
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
