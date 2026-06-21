package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestListGetDeleteKey(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	c := NewClient(mr.Addr())
	ctx := context.Background()

	if err := c.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	mr.Set("foo", "bar")
	mr.Set("other", "x")
	mr.SetTTL("other", 30*time.Second)

	keys, err := c.ListKeys(ctx, "f*")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Key != "foo" {
		t.Fatalf("list: %+v", keys)
	}
	if keys[0].TTL != -1 && keys[0].TTL != 0 {
		t.Fatalf("foo ttl: %d", keys[0].TTL)
	}

	detail, err := c.GetKey(ctx, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Type != "string" || detail.Value != "bar" {
		t.Fatalf("get: %+v", detail)
	}

	if err := c.DeleteKey(ctx, "foo"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteKey(ctx, "foo"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
