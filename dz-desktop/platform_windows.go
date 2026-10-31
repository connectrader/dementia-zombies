//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

func udpControl(network, address string, c syscall.RawConn) error {
	var serr error
	c.Control(func(fd uintptr) {
		h := syscall.Handle(fd)
		syscall.SetsockoptInt(h, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		serr = syscall.SetsockoptInt(h, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	})
	return serr
}

func browserCandidates() []string {
	var out []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LocalAppData"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		out = append(out,
			filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(base, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		)
	}
	return out
}

// openWindow opens the game in its own app window (no tabs or address bar) using Edge or Chrome,
// with a private profile so saves and settings stay with the game. The channel closes when that window's process exits.
func openWindow(url, profile string) chan struct{} {
	done := make(chan struct{})
	for _, exe := range browserCandidates() {
		if _, err := os.Stat(exe); err != nil {
			continue
		}
		cmd := exec.Command(exe, "--app="+url, "--user-data-dir="+profile, "--no-first-run", "--no-default-browser-check",
			"--window-size=1280,820", "--autoplay-policy=no-user-gesture-required", "--hide-crash-restore-bubble",
			"--disable-features=Translate,msEdgeSidebarV2,msHubApps", "--disable-background-timer-throttling",
			"--disable-renderer-backgrounding", "--disable-backgrounding-occluded-windows")
		if err := cmd.Start(); err != nil {
			logger.Printf("start %s: %v", exe, err)
			continue
		}
		logger.Printf("opened window with %s", exe)
		go func() { cmd.Wait(); close(done) }()
		return done
	}
	logger.Printf("no Edge or Chrome found, using the default browser")
	exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	close(done)
	return done
}

func showError(msg string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	box := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(msg)
	c, _ := syscall.UTF16PtrFromString("Dementia Zombies")
	box.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
}

// detach starts a child that outlives this process, with no console window.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200} // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
}
