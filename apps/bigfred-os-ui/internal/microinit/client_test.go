package microinit_test

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/microinit"
)

func TestClientListAndControl(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "microinit.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleFake(conn)
		}
	}()

	c := &microinit.Client{Socket: sock, Timeout: 2 * time.Second}
	list, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "redis" || list[0].State != "running" {
		t.Fatalf("list: %+v", list)
	}
	if err := c.Control("redis", "restart"); err != nil {
		t.Fatal(err)
	}
	if err := c.Control("redis", "pause"); err == nil {
		t.Fatal("expected invalid action")
	}
}

func handleFake(conn net.Conn) {
	defer conn.Close()
	req, err := readReq(conn)
	if err != nil {
		return
	}
	switch req.Type {
	case "list":
		_ = writeResp(conn, map[string]any{
			"type": "list",
			"services": []map[string]any{{
				"name": "redis", "state": "running", "pid": 42,
				"restarts": 0, "enabled": true,
			}},
		})
	case "start", "stop", "restart":
		_ = writeResp(conn, map[string]any{"type": "ok"})
	case "logs":
		_ = writeResp(conn, map[string]any{
			"type": "log",
			"line": map[string]any{
				"ts": "t", "service": req.Name, "level": "stdout", "msg": "hello",
			},
		})
		if req.Follow != nil && !*req.Follow {
			_ = writeResp(conn, map[string]any{"type": "ok"})
		}
	default:
		_ = writeResp(conn, map[string]any{"type": "error", "message": "unknown"})
	}
}

type fakeReq struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Follow *bool  `json:"follow"`
	Lines  *uint64 `json:"lines"`
}

func readReq(r io.Reader) (fakeReq, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return fakeReq{}, err
	}
	n := binary.LittleEndian.Uint32(hdr[:])
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return fakeReq{}, err
	}
	var req fakeReq
	err := json.Unmarshal(buf, &req)
	return req, err
}

func writeResp(w io.Writer, msg any) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}
