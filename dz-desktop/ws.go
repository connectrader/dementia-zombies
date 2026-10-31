package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
)

// A small RFC 6455 WebSocket server: text frames, fragmentation, ping and close. Enough for presence relay.

const maxMsg = 4 << 20

func wsUpgrade(w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.ReadWriter, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || r.Header.Get("Sec-WebSocket-Key") == "" {
		http.Error(w, "websocket only", http.StatusBadRequest)
		return nil, nil, errors.New("not websocket")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "no hijack", http.StatusInternalServerError)
		return nil, nil, errors.New("no hijack")
	}
	sum := sha1.Sum([]byte(r.Header.Get("Sec-WebSocket-Key") + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	conn, brw, err := hj.Hijack()
	if err != nil {
		return nil, nil, err
	}
	brw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " +
		base64.StdEncoding.EncodeToString(sum[:]) + "\r\n\r\n")
	if err := brw.Flush(); err != nil {
		conn.Close()
		return nil, nil, err
	}
	return conn, brw, nil
}

func wsFrame(br *bufio.Reader) (fin bool, op byte, data []byte, err error) {
	var hdr [2]byte
	if _, err = io.ReadFull(br, hdr[:]); err != nil {
		return
	}
	fin, op = hdr[0]&0x80 != 0, hdr[0]&0x0f
	masked := hdr[1]&0x80 != 0
	n := uint64(hdr[1] & 0x7f)
	switch n {
	case 126:
		var b [2]byte
		if _, err = io.ReadFull(br, b[:]); err != nil {
			return
		}
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		if _, err = io.ReadFull(br, b[:]); err != nil {
			return
		}
		n = binary.BigEndian.Uint64(b[:])
	}
	if n > maxMsg {
		err = errors.New("frame too big")
		return
	}
	var key [4]byte
	if masked {
		if _, err = io.ReadFull(br, key[:]); err != nil {
			return
		}
	}
	data = make([]byte, n)
	if _, err = io.ReadFull(br, data); err != nil {
		return
	}
	if masked {
		for i := range data {
			data[i] ^= key[i&3]
		}
	}
	return
}

// wsRead returns one whole message (continuation frames joined) or a control frame.
func wsRead(br *bufio.Reader) (byte, []byte, error) {
	var buf []byte
	var op byte
	for {
		fin, o, data, err := wsFrame(br)
		if err != nil {
			return 0, nil, err
		}
		if o >= 8 {
			return o, data, nil
		}
		if o != 0 {
			op = o
		}
		buf = append(buf, data...)
		if len(buf) > maxMsg {
			return 0, nil, errors.New("message too big")
		}
		if fin {
			return op, buf, nil
		}
	}
}

func wsWrite(w io.Writer, op byte, data []byte) error {
	hdr := make([]byte, 0, 10)
	hdr = append(hdr, 0x80|op)
	switch n := len(data); {
	case n < 126:
		hdr = append(hdr, byte(n))
	case n <= 0xffff:
		hdr = append(hdr, 126, byte(n>>8), byte(n))
	default:
		hdr = append(hdr, 127)
		hdr = binary.BigEndian.AppendUint64(hdr, uint64(n))
	}
	if _, err := w.Write(hdr); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}
