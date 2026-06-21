package redis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestWatchKey(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	c := NewClient(mr.Addr())
	mr.Set("live", "v1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var events []WatchKind
	done := make(chan struct{})

	go func() {
		defer close(done)
		_ = c.WatchKey(ctx, "live", func(ev WatchEvent) {
			mu.Lock()
			events = append(events, ev.Kind)
			mu.Unlock()
		})
	}()

	waitFor := func(kind WatchKind, timeout time.Duration) {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			mu.Lock()
			ok := false
			for _, k := range events {
				if k == kind {
					ok = true
					break
				}
			}
			mu.Unlock()
			if ok {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("timeout waiting for %s, got %v", kind, events)
	}

	waitFor(WatchSnapshot, time.Second)

	mr.Set("live", "v2")
	waitFor(WatchUpdate, 2*time.Second)

	mr.Del("live")
	waitFor(WatchDeleted, 2*time.Second)

	cancel()
	<-done
}
