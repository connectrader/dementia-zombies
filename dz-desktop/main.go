// Dementia Zombies desktop: serves the game to a local app window and relays LAN co-op.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed web/game.html
var gameHTML []byte

const (
	basePort = 47800
	discPort = 47801
)

// version is stamped in at build time from GAME_VERSION in the game file
var version = "dev"

var (
	solo       bool // the usual single copy: shared saves and window profile
	port       int
	instanceID = randID()
	dataDir    string
	logger     *log.Logger
	started    = time.Now()
)

type safeWriter struct{ ws []io.Writer }

func (s safeWriter) Write(p []byte) (int, error) {
	for _, w := range s.ws {
		if w != nil {
			w.Write(p)
		}
	}
	return len(p), nil
}

func main() {
	noOpen := flag.Bool("no-open", false, "run as a server only, without opening a window")
	fixed := flag.Int("port", 0, "port to use (default 47800, or the next free one)")
	afterUpd := flag.Bool("after-update", false, "started by the previous version after an update")
	flag.Parse()

	dataDir = appDataDir()
	os.MkdirAll(dataDir, 0o755)
	lf, _ := os.OpenFile(filepath.Join(dataDir, "log.txt"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	logger = log.New(safeWriter{[]io.Writer{os.Stderr, lf}}, "", log.LstdFlags)

	// one copy at a time: if the game is already running, open another window onto it and step aside
	go cleanOld()
	if *afterUpd {
		// take over from the version that just updated us; its window is still open and will reconnect
		waitPort(*fixed)
	} else if *fixed == 0 && handOff() {
		return
	}
	solo = *fixed == 0 || (*afterUpd && *fixed == basePort)

	ln, err := listen(*fixed)
	if err != nil {
		fatal("Dementia Zombies couldn't open a network port.\n\n" + err.Error())
		return
	}
	port = ln.Addr().(*net.TCPAddr).Port
	logger.Printf("Dementia Zombies %s on port %d, data in %s, LAN %v", version, port, dataDir, lanIPs())

	hub := newHub()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(gameHTML)
	})
	mux.HandleFunc("/ws", hub.serveWS)
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		writeJSON(w, map[string]any{"app": "dz", "version": version, "port": port, "ips": lanIPs()})
	})
	mux.HandleFunc("/api/games", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		writeJSON(w, map[string]any{"app": "dz", "id": instanceID, "port": port, "games": hub.localGames()})
	})
	mux.HandleFunc("/api/addhost", func(w http.ResponseWriter, r *http.Request) {
		if !trusted(r) || r.Method != http.MethodPost {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		addr, ok := normAddr(r.URL.Query().Get("addr"))
		if !ok {
			writeJSON(w, map[string]any{"ok": false, "err": "bad address"})
			return
		}
		n, err := hub.pollHost(addr)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "err": err.Error()})
			return
		}
		hub.addManual(addr)
		writeJSON(w, map[string]any{"ok": true, "games": n})
	})
	mux.HandleFunc("/api/handoff", func(w http.ResponseWriter, r *http.Request) {
		if !trusted(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		hub.pokeNow()
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/update/check", handleUpdCheck)
	mux.HandleFunc("/api/update/install", handleUpdInstall)
	mux.HandleFunc("/api/update/restart", handleUpdRestart)
	mux.HandleFunc("/api/save/", handleSave)
	mux.HandleFunc("/api/export", handleExport)

	go func() {
		srv := &http.Server{Handler: hostGuard(mux), ReadHeaderTimeout: 10 * time.Second}
		if err := srv.Serve(ln); err != nil {
			logger.Printf("server stopped: %v", err)
		}
	}()
	go discovery(hub)

	if *noOpen {
		fmt.Printf("Dementia Zombies server running.\n  This computer: http://127.0.0.1:%d/\n", port)
		for _, ip := range lanIPs() {
			fmt.Printf("  On your network: http://%s:%d/\n", ip, port)
		}
		select {}
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	profile := windowProfile(port)
	var winDone chan struct{}
	if *afterUpd {
		winDone = make(chan struct{})
		close(winDone) // the window belongs to the old version; stay up while it's connected
	} else {
		winDone = openWindow(url, profile)
	}

	// Quit once the game window is gone: the browser process has exited (or we couldn't track it)
	// and no page on this computer has been connected for a few seconds.
	var idleSince time.Time
	for range time.Tick(time.Second) {
		closed := false
		select {
		case <-winDone:
			closed = true
		default:
		}
		if hub.localClients() > 0 || hub.sincePoke() < 30*time.Second {
			idleSince = time.Time{}
			continue
		}
		if idleSince.IsZero() {
			idleSince = time.Now()
		}
		grace := 8 * time.Second
		if !closed && time.Since(started) < 120*time.Second {
			continue // the window may still be starting up
		}
		if !hub.everLocal() && time.Since(started) < 120*time.Second {
			continue
		}
		if time.Since(idleSince) > grace {
			logger.Printf("window closed, exiting")
			os.Exit(0)
		}
	}
}

func listen(fixed int) (net.Listener, error) {
	if fixed > 0 {
		return net.Listen("tcp", fmt.Sprintf(":%d", fixed))
	}
	var last error
	for p := basePort; p < basePort+10; p++ {
		if p == discPort {
			continue
		}
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err == nil {
			return ln, nil
		}
		last = err
	}
	return nil, last
}

func appDataDir() string {
	if d := os.Getenv("APPDATA"); d != "" {
		return filepath.Join(d, "Dementia Zombies")
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "dementia-zombies")
	}
	return filepath.Join(os.TempDir(), "dementia-zombies")
}

func windowProfile(p int) string {
	base := dataDir
	if d := os.Getenv("LOCALAPPDATA"); d != "" {
		base = filepath.Join(d, "Dementia Zombies")
	}
	if solo || p == basePort {
		return filepath.Join(base, "window")
	}
	return filepath.Join(base, "window-"+strconv.Itoa(p))
}

// handOff asks a copy that's already running to keep going, opens a window onto it, and reports whether it did.
func handOff() bool {
	c := &http.Client{Timeout: 1500 * time.Millisecond}
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/api/handoff", basePort), nil)
	req.Header.Set("X-DZ", "1")
	resp, err := c.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/", basePort)
	solo = true
	openWindow(url, windowProfile(basePort))
	return true
}

func saveDir() string {
	if solo || port == basePort {
		return filepath.Join(dataDir, "saves")
	}
	return filepath.Join(dataDir, "saves-"+strconv.Itoa(port))
}

var idRx = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)

var slotMu sync.Map

func slotLock(id string) *sync.Mutex {
	m, _ := slotMu.LoadOrStore(id, &sync.Mutex{})
	return m.(*sync.Mutex)
}

func handleSave(w http.ResponseWriter, r *http.Request) {
	if !trusted(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/save/")
	if !idRx.MatchString(id) {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	dir := saveDir()
	os.MkdirAll(dir, 0o755)
	fn := filepath.Join(dir, id+".json")
	switch r.Method {
	case http.MethodGet:
		b, err := os.ReadFile(fn)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	case http.MethodPut:
		b, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
		if err != nil || !json.Valid(b) {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		lk := slotLock(id)
		lk.Lock()
		defer lk.Unlock()
		tmp := fn + "." + randID() + ".tmp"
		if err := os.WriteFile(tmp, b, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var rerr error
		for i := 0; i < 8; i++ { // virus scanners briefly hold new files open on Windows
			if rerr = os.Rename(tmp, fn); rerr == nil {
				break
			}
			time.Sleep(time.Duration(50*(i+1)) * time.Millisecond)
		}
		if rerr != nil {
			os.Remove(tmp)
			http.Error(w, rerr.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	case http.MethodDelete:
		lk := slotLock(id)
		lk.Lock()
		os.Remove(fn)
		lk.Unlock()
		writeJSON(w, map[string]any{"ok": true})
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

var nameRx = regexp.MustCompile(`[^A-Za-z0-9._ -]`)

func handleExport(w http.ResponseWriter, r *http.Request) {
	if !trusted(r) || r.Method != http.MethodPost {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	name := strings.Trim(nameRx.ReplaceAllString(r.URL.Query().Get("name"), ""), ". ")
	if name == "" {
		name = "dementia-zombies-save"
	}
	if len(name) > 80 {
		name = name[:80]
	}
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dir := exportDir()
	os.MkdirAll(dir, 0o755)
	fn := filepath.Join(dir, name)
	if err := os.WriteFile(fn, b, 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "path": fn})
}

func exportDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		for _, d := range []string{"Documents", "OneDrive/Documents"} {
			if st, err := os.Stat(filepath.Join(h, d)); err == nil && st.IsDir() {
				return filepath.Join(h, d, "Dementia Zombies")
			}
		}
	}
	return filepath.Join(dataDir, "exports")
}

func isLoopback(r *http.Request) bool {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// hostGuard refuses requests addressed by domain name, which blocks DNS-rebinding tricks from websites.
func hostGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostLiteral(r.Host) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func hostLiteral(hp string) bool {
	h, _, err := net.SplitHostPort(hp)
	if err != nil {
		h = hp
	}
	h = strings.Trim(h, "[]")
	return strings.EqualFold(h, "localhost") || net.ParseIP(h) != nil
}

// originOK lets the game's own pages in (served by any copy of the app, addressed by IP) and keeps websites out.
func originOK(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	u, err := url.Parse(o)
	if err != nil || u.Scheme != "http" {
		return false
	}
	return hostLiteral(u.Host)
}

// trusted is for endpoints that touch this computer's files: only the game window on this computer,
// sending a custom header (which a website can't send cross-site without a preflight we never grant).
func trusted(r *http.Request) bool {
	if !isLoopback(r) || r.Header.Get("X-DZ") != "1" || !originOK(r) {
		return false
	}
	h, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		h = r.Host
	}
	return h == "127.0.0.1" || strings.EqualFold(h, "localhost") || h == "[::1]" || h == "::1"
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-store")
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// normAddr accepts "192.168.1.20", "192.168.1.20:47800", "http://host:port/" and returns host:port.
func normAddr(s string) (string, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "http://"), "https://")
	s = strings.TrimSuffix(strings.SplitN(s, "/", 2)[0], "/")
	if s == "" || len(s) > 100 {
		return "", false
	}
	host, p, err := net.SplitHostPort(s)
	if err != nil {
		host, p = s, strconv.Itoa(basePort)
	}
	if host == "" || strings.ContainsAny(host, " @") {
		return "", false
	}
	if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
		return "", false
	}
	return net.JoinHostPort(host, p), true
}

func lanIPs() []string {
	var out []string
	ifs, _ := net.Interfaces()
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				if ip4 := ipn.IP.To4(); ip4 != nil && !ip4.IsLinkLocalUnicast() {
					out = append(out, ip4.String())
				}
			}
		}
	}
	// private ranges first: those are the ones friends on the same Wi-Fi can reach
	priv, rest := []string{}, []string{}
	for _, ip := range out {
		if net.ParseIP(ip).IsPrivate() {
			priv = append(priv, ip)
		} else {
			rest = append(rest, ip)
		}
	}
	return append(priv, rest...)
}

func fatal(msg string) {
	if logger != nil {
		logger.Print(msg)
	}
	showError(msg)
	os.Exit(1)
}
