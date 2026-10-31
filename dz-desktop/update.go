package main

// Self-update from GitHub releases.
// A release is a tag vX.Y.Z with two assets: DementiaZombies-vX.Y.Z-windows-x64.exe and SHA256SUMS.txt.
// The game asks /api/update/check, the player confirms, /api/update/install downloads and verifies the new exe,
// swaps it in next to the running one (Windows allows renaming a running exe), then /api/update/restart starts it.

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const updRepo = "connectrader/dementia-zombies"

type relAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	API  string `json:"url"`
	Size int64  `json:"size"`
}
type release struct {
	Tag       string     `json:"tag_name"`
	Name      string     `json:"name"`
	Body      string     `json:"body"`
	Published string     `json:"published_at"`
	Draft     bool       `json:"draft"`
	Pre       bool       `json:"prerelease"`
	Assets    []relAsset `json:"assets"`
}

// exePath is where this program was started from, recorded before an update renames it.
var exePath = func() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if q, err := filepath.EvalSymlinks(p); err == nil {
		p = q
	}
	return p
}()

var (
	updMu     sync.Mutex
	updCache  *release
	updAt     time.Time
	updClient = &http.Client{Timeout: 20 * time.Second}
	dlClient  = &http.Client{Timeout: 10 * time.Minute}
)

func semver(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	var out [3]int
	for i, p := range strings.SplitN(v, ".", 3) {
		n, _ := strconv.Atoi(strings.TrimFunc(p, func(r rune) bool { return r < '0' || r > '9' }))
		out[i] = n
	}
	return out
}

func newer(a, b string) bool { // is a newer than b
	x, y := semver(a), semver(b)
	for i := 0; i < 3; i++ {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

func ghGet(url string, accept string, c *http.Client) (*http.Response, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "DementiaZombies/"+version)
	req.Header.Set("Accept", accept)
	if t := os.Getenv("DZ_GITHUB_TOKEN"); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	return c.Do(req)
}

func latestRelease(force bool) (*release, error) {
	updMu.Lock()
	defer updMu.Unlock()
	if !force && updCache != nil && time.Since(updAt) < 10*time.Minute {
		return updCache, nil
	}
	api := "https://api.github.com"
	if v := os.Getenv("DZ_UPDATE_API"); v != "" { // for testing against a local stand-in
		api = strings.TrimRight(v, "/")
	}
	resp, err := ghGet(api+"/repos/"+updRepo+"/releases/latest", "application/vnd.github+json", updClient)
	if err != nil {
		return nil, errors.New("couldn't reach GitHub")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, errors.New("GitHub has no public releases of the game to check (the repository may be private)")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub answered %d", resp.StatusCode)
	}
	var r release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&r); err != nil {
		return nil, errors.New("couldn't read the release")
	}
	updCache, updAt = &r, time.Now()
	return &r, nil
}

func (r *release) asset(suffix string) *relAsset {
	for i := range r.Assets {
		if strings.HasSuffix(strings.ToLower(r.Assets[i].Name), suffix) {
			return &r.Assets[i]
		}
	}
	return nil
}

func handleUpdCheck(w http.ResponseWriter, r *http.Request) {
	if !trusted(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	rel, err := latestRelease(r.URL.Query().Get("force") == "1")
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "current": version, "err": err.Error()})
		return
	}
	exe := rel.asset("-windows-x64.exe")
	notes := rel.Body
	if len(notes) > 3000 {
		notes = notes[:3000] + "…"
	}
	writeJSON(w, map[string]any{"ok": true, "current": version, "latest": strings.TrimPrefix(rel.Tag, "v"), "newer": newer(rel.Tag, version) && exe != nil,
		"name": rel.Name, "notes": notes, "published": rel.Published, "size": func() int64 {
			if exe != nil {
				return exe.Size
			}
			return 0
		}(), "installable": canSelfUpdate()})
}

func canSelfUpdate() bool {
	if version == "dev" {
		return false
	}
	exe := exePath
	if exe == "" {
		return false
	}
	f, err := os.CreateTemp(filepath.Dir(exe), ".dzwrite-*")
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(f.Name())
	return true
}

var installing sync.Mutex

func handleUpdInstall(w http.ResponseWriter, r *http.Request) {
	if !trusted(r) || r.Method != http.MethodPost {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !installing.TryLock() {
		writeJSON(w, map[string]any{"ok": false, "err": "an update is already installing"})
		return
	}
	defer installing.Unlock()
	if err := installUpdate(); err != nil {
		logger.Printf("update failed: %v", err)
		writeJSON(w, map[string]any{"ok": false, "err": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func installUpdate() error {
	rel, err := latestRelease(true)
	if err != nil {
		return err
	}
	if !newer(rel.Tag, version) {
		return errors.New("already up to date")
	}
	exeA, sumA := rel.asset("-windows-x64.exe"), rel.asset("sha256sums.txt")
	if exeA == nil || sumA == nil {
		return errors.New("the release is missing its files")
	}
	// checksum list
	resp, err := download(sumA)
	if err != nil {
		return err
	}
	want := ""
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 1<<20))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == exeA.Name {
			want = strings.ToLower(f[0])
		}
	}
	resp.Body.Close()
	if len(want) != 64 {
		return errors.New("no checksum for the new version")
	}
	exe := exePath
	if exe == "" {
		return errors.New("can't find the game's own file")
	}
	tmp := exe + ".download"
	os.Remove(tmp)
	resp, err = download(exeA)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		resp.Body.Close()
		return errors.New("can't write next to the game: " + err.Error())
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, 200<<20))
	resp.Body.Close()
	f.Close()
	if err != nil || (exeA.Size > 0 && n != exeA.Size) {
		os.Remove(tmp)
		return errors.New("the download was cut off")
	}
	if hex.EncodeToString(h.Sum(nil)) != want {
		os.Remove(tmp)
		return errors.New("the download didn't match its checksum")
	}
	if b, err := os.ReadFile(tmp); err != nil || len(b) < 2 || (runtime.GOOS == "windows" && (b[0] != 'M' || b[1] != 'Z')) {
		os.Remove(tmp)
		return errors.New("the download isn't a Windows program")
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(tmp)
		return errors.New("couldn't move the old version aside: " + err.Error())
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Rename(old, exe)
		return errors.New("couldn't put the new version in place: " + err.Error())
	}
	logger.Printf("updated %s -> %s", version, rel.Tag)
	return nil
}

func download(a *relAsset) (*http.Response, error) {
	url, accept := a.URL, "application/octet-stream"
	if os.Getenv("DZ_GITHUB_TOKEN") != "" && a.API != "" {
		url = a.API // private repos serve assets through the API
	}
	resp, err := ghGet(url, accept, dlClient)
	if err != nil {
		return nil, errors.New("couldn't download the update")
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed (%d)", resp.StatusCode)
	}
	return resp, nil
}

// restart: start the new version and step aside. It takes over this port, and the open game window reconnects to it.
func handleUpdRestart(w http.ResponseWriter, r *http.Request) {
	if !trusted(r) || r.Method != http.MethodPost {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	exe := exePath
	cmd := exec.Command(exe, "-after-update", "-port", strconv.Itoa(port))
	cmd.Dir = filepath.Dir(exe)
	detach(cmd)
	if err := cmd.Start(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "err": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
	go func() { time.Sleep(400 * time.Millisecond); os.Exit(0) }()
}

// cleanOld removes the previous version left behind by an update.
func cleanOld() {
	if exePath == "" {
		return
	}
	old := exePath + ".old"
	for i := 0; i < 20; i++ {
		if err := os.Remove(old); err == nil || os.IsNotExist(err) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// waitPort waits for the previous version to let go of the port.
func waitPort(p int) {
	for i := 0; i < 40; i++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err == nil {
			ln.Close()
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
}
