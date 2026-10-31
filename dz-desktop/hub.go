package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"sync"
	"time"
)

// A room is a set of peers sharing presence objects, the same model the game uses on claude.ai:
// every peer publishes one JSON object, and everyone else sees the latest copy.

const lobbyRoom = "__lobby"

// peer outbox: only the newest message per key is kept, so a slow connection skips stale snapshots
// instead of falling behind, and a "left" message replaces that peer's last update rather than being dropped.
type peer struct {
	id    string
	room  string
	local bool
	pres  json.RawMessage
	conn  net.Conn
	once  sync.Once

	qmu   sync.Mutex
	pend  map[string][]byte
	order []string
	wake  chan struct{}
	done  chan struct{}
}

func (p *peer) send(key string, b []byte) {
	p.qmu.Lock()
	if _, ok := p.pend[key]; !ok {
		p.order = append(p.order, key)
	}
	p.pend[key] = b
	p.qmu.Unlock()
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *peer) take() [][]byte {
	p.qmu.Lock()
	defer p.qmu.Unlock()
	out := make([][]byte, 0, len(p.order))
	for _, k := range p.order {
		out = append(out, p.pend[k])
	}
	p.pend, p.order = map[string][]byte{}, p.order[:0]
	return out
}

func (p *peer) close() { p.once.Do(func() { close(p.done); p.conn.Close() }) }

type game struct {
	Code string `json:"code"`
	N    string `json:"n"`
	Lv   int64  `json:"lv"`
	C    int    `json:"c"`
}

type remoteHost struct {
	games  []game
	seen   time.Time
	manual bool
}

type hub struct {
	mu      sync.Mutex
	rooms   map[string]map[string]*peer
	virt    map[string]json.RawMessage // lobby entries for games on other computers
	remotes map[string]*remoteHost     // host:port -> games
	manual  map[string]bool
	everLoc bool
	poke    time.Time
	client  *http.Client
}

func newHub() *hub {
	return &hub{rooms: map[string]map[string]*peer{}, virt: map[string]json.RawMessage{}, remotes: map[string]*remoteHost{}, manual: map[string]bool{},
		client: &http.Client{Timeout: 2 * time.Second}}
}

func randID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func msg(v any) []byte { b, _ := json.Marshal(v); return b }

var peerIDRx = regexp.MustCompile(`^[0-9a-f]{12}$`)

func (h *hub) serveWS(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	if room == "" || len(room) > 64 {
		http.Error(w, "room", http.StatusBadRequest)
		return
	}
	if !originOK(r) {
		http.Error(w, "origin", http.StatusForbidden)
		return
	}
	conn, brw, err := wsUpgrade(w, r)
	if err != nil {
		return
	}
	p := &peer{room: room, local: isLoopback(r), pres: json.RawMessage("{}"), conn: conn,
		pend: map[string][]byte{}, wake: make(chan struct{}, 1), done: make(chan struct{})}

	h.mu.Lock()
	rm := h.rooms[room]
	if rm == nil {
		rm = map[string]*peer{}
		h.rooms[room] = rm
	}
	// a reconnecting page keeps its id, so the host doesn't see it leave and rejoin
	if want := r.URL.Query().Get("id"); peerIDRx.MatchString(want) {
		if old, taken := rm[want]; !taken {
			p.id = want
		} else if old.local == p.local {
			old.close()
			p.id = want
		}
	}
	if p.id == "" {
		p.id = randID()
	}
	list := []map[string]any{}
	for _, q := range rm {
		if q.id != p.id {
			list = append(list, map[string]any{"peer": q.id, "p": q.pres})
		}
	}
	if room == lobbyRoom {
		for id, pr := range h.virt {
			list = append(list, map[string]any{"peer": id, "p": pr})
		}
		if p.local {
			h.everLoc = true
		}
	}
	rm[p.id] = p
	p.send("hello", msg(map[string]any{"t": "hello", "you": p.id, "peers": list}))
	h.mu.Unlock()

	go p.writer(conn)

	br := brw.Reader
	for {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		op, data, err := wsRead(br)
		if err != nil {
			break
		}
		if op == 8 {
			p.send("close", nil)
			break
		}
		if op != 1 {
			continue
		}
		var m struct {
			T string          `json:"t"`
			P json.RawMessage `json:"p"`
		}
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if m.T == "p" && len(m.P) > 0 && m.P[0] == '{' {
			h.mu.Lock()
			if rm[p.id] == p {
				p.pres = m.P
				out := msg(map[string]any{"t": "u", "peer": p.id, "p": m.P})
				for _, q := range rm {
					if q != p {
						q.send("p:"+p.id, out)
					}
				}
			}
			h.mu.Unlock()
		}
	}
	time.Sleep(50 * time.Millisecond) // let a close reply go out
	p.close()
	h.mu.Lock()
	if rm[p.id] == p {
		delete(rm, p.id)
		out := msg(map[string]any{"t": "x", "peer": p.id})
		for _, q := range rm {
			q.send("p:"+p.id, out)
		}
	}
	if len(rm) == 0 && h.rooms[room] != nil && len(h.rooms[room]) == 0 {
		delete(h.rooms, room)
	}
	h.mu.Unlock()
}

func (p *peer) writer(conn net.Conn) {
	bw := bufio.NewWriterSize(conn, 32<<10)
	ping := time.NewTicker(5 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-p.done:
			return
		case <-p.wake:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			for _, b := range p.take() {
				var err error
				if b == nil {
					err = wsWrite(bw, 8, []byte{0x03, 0xe8}) // close, 1000
				} else {
					err = wsWrite(bw, 1, b)
				}
				if err != nil {
					p.close()
					return
				}
			}
			if bw.Flush() != nil {
				p.close()
				return
			}
		case <-ping.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if wsWrite(bw, 1, []byte(`{"t":"k"}`)) != nil || bw.Flush() != nil {
				p.close()
				return
			}
		}
	}
}

func (h *hub) localClients() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, q := range h.rooms[lobbyRoom] {
		if q.local {
			n++
		}
	}
	return n
}

func (h *hub) everLocal() bool { h.mu.Lock(); defer h.mu.Unlock(); return h.everLoc }

// pokeNow records that another copy of the app just handed a window over to this one.
func (h *hub) pokeNow() { h.mu.Lock(); h.poke = time.Now(); h.mu.Unlock() }

func (h *hub) sincePoke() time.Duration { h.mu.Lock(); defer h.mu.Unlock(); return time.Since(h.poke) }

// localGames lists the games hosted by players connected to this server.
func (h *hub) localGames() []game {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := []game{}
	for _, q := range h.rooms[lobbyRoom] {
		var pr struct {
			Dz *game `json:"dz"`
		}
		if json.Unmarshal(q.pres, &pr) == nil && pr.Dz != nil && pr.Dz.Code != "" {
			if _, live := h.rooms["dz-"+pr.Dz.Code]; live {
				out = append(out, *pr.Dz)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

func (h *hub) setRemote(addr string, games []game, manual bool) {
	h.mu.Lock()
	r := h.remotes[addr]
	if r == nil {
		r = &remoteHost{}
		h.remotes[addr] = r
	}
	r.games, r.seen = games, time.Now()
	r.manual = r.manual || manual
	h.mu.Unlock()
}

func (h *hub) addManual(addr string) { h.mu.Lock(); h.manual[addr] = true; h.mu.Unlock() }

func (h *hub) pollHost(addr string) (int, error) {
	resp, err := h.client.Get("http://" + addr + "/api/games")
	if err != nil {
		return 0, fmt.Errorf("unreachable")
	}
	defer resp.Body.Close()
	var g struct {
		App   string `json:"app"`
		ID    string `json:"id"`
		Games []game `json:"games"`
	}
	if json.NewDecoder(resp.Body).Decode(&g) != nil || g.App != "dz" {
		return 0, fmt.Errorf("not a Dementia Zombies game")
	}
	if g.ID == instanceID {
		return len(g.Games), nil // that's us
	}
	if len(g.Games) > 16 {
		g.Games = g.Games[:16]
	}
	h.setRemote(addr, g.Games, true)
	return len(g.Games), nil
}

// refreshVirtual rebuilds the lobby's list of games on other computers and tells local pages what changed.
func (h *hub) refreshVirtual() {
	h.mu.Lock()
	defer h.mu.Unlock()
	localCodes := map[string]bool{}
	for _, q := range h.rooms[lobbyRoom] {
		var pr struct {
			Dz *game `json:"dz"`
		}
		if json.Unmarshal(q.pres, &pr) == nil && pr.Dz != nil {
			localCodes[pr.Dz.Code] = true
		}
	}
	next := map[string]json.RawMessage{}
	for addr, r := range h.remotes {
		ttl := 5 * time.Second
		if r.manual {
			ttl = 8 * time.Second
		}
		if time.Since(r.seen) > ttl {
			delete(h.remotes, addr)
			continue
		}
		for _, g := range r.games {
			if g.Code == "" || len(g.Code) > 8 || localCodes[g.Code] {
				continue
			}
			id := "lan:" + addr + ":" + g.Code
			next[id] = msg(map[string]any{"dz": map[string]any{"code": g.Code, "n": g.N, "lv": g.Lv, "c": g.C, "addr": addr}})
		}
	}
	type upd struct {
		id string
		b  []byte
	}
	var outs []upd
	for id, pr := range next {
		if old, ok := h.virt[id]; !ok || string(old) != string(pr) {
			outs = append(outs, upd{id, msg(map[string]any{"t": "u", "peer": id, "p": pr})})
		}
	}
	for id := range h.virt {
		if _, ok := next[id]; !ok {
			outs = append(outs, upd{id, msg(map[string]any{"t": "x", "peer": id})})
		}
	}
	h.virt = next
	for _, q := range h.rooms[lobbyRoom] {
		for _, u := range outs {
			q.send("p:"+u.id, u.b)
		}
	}
}

func (h *hub) manualHosts() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := []string{}
	for a := range h.manual {
		out = append(out, a)
	}
	return out
}
