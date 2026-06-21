package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const watchPollInterval = 400 * time.Millisecond

// WatchKind classifies key watch events.
type WatchKind string

const (
	WatchSnapshot WatchKind = "snapshot"
	WatchUpdate   WatchKind = "update"
	WatchDeleted  WatchKind = "deleted"
)

// WatchEvent is emitted while watching a single key.
type WatchEvent struct {
	Kind   WatchKind
	Detail KeyDetail
}

// WatchKey streams snapshot and update events until ctx is cancelled.
func (c *Client) WatchKey(ctx context.Context, key string, emit func(WatchEvent)) error {
	if key == "" {
		return fmt.Errorf("empty key")
	}

	detail, err := c.GetKey(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			emit(WatchEvent{Kind: WatchDeleted})
		} else {
			return err
		}
	} else {
		emit(WatchEvent{Kind: WatchSnapshot, Detail: detail})
	}

	if c.ensureKeyspaceNotifications(ctx) == nil {
		return c.watchKeyspace(ctx, key, emit)
	}
	var initial *KeyDetail
	if err == nil {
		initial = &detail
	}
	return c.watchPoll(ctx, key, emit, initial)
}

func (c *Client) watchKeyspace(ctx context.Context, key string, emit func(WatchEvent)) error {
	pubsub := c.rdb.Subscribe(ctx, keyspaceChannel(key))
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-pubsub.Channel():
			if !ok {
				return nil
			}
			if isDeleteEvent(msg.Payload) {
				emit(WatchEvent{Kind: WatchDeleted})
				continue
			}
			detail, err := c.GetKey(ctx, key)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					emit(WatchEvent{Kind: WatchDeleted})
					continue
				}
				return err
			}
			emit(WatchEvent{Kind: WatchUpdate, Detail: detail})
		}
	}
}

func (c *Client) watchPoll(ctx context.Context, key string, emit func(WatchEvent), initial *KeyDetail) error {
	last := ""
	if initial != nil {
		last = keyFingerprint(*initial)
	}

	ticker := time.NewTicker(watchPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			detail, err := c.GetKey(ctx, key)
			if errors.Is(err, ErrNotFound) {
				if last != "" {
					emit(WatchEvent{Kind: WatchDeleted})
					last = ""
				}
				continue
			}
			if err != nil {
				return err
			}
			fp := keyFingerprint(detail)
			if fp == last {
				continue
			}
			kind := WatchUpdate
			if last == "" {
				kind = WatchSnapshot
			}
			emit(WatchEvent{Kind: kind, Detail: detail})
			last = fp
		}
	}
}

func keyFingerprint(d KeyDetail) string {
	payload := struct {
		Type  string `json:"type"`
		TTL   int64  `json:"ttl"`
		Value any    `json:"value"`
	}{d.Type, d.TTL, d.Value}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("%s:%d:%v", d.Type, d.TTL, d.Value)
	}
	return string(b)
}

func keyspaceChannel(key string) string {
	return "__keyspace@0__:" + key
}

func isDeleteEvent(payload string) bool {
	switch payload {
	case "del", "expired", "evicted", "unlink":
		return true
	default:
		return false
	}
}

func (c *Client) ensureKeyspaceNotifications(ctx context.Context) error {
	flags, err := c.rdb.ConfigGet(ctx, "notify-keyspace-events").Result()
	if err != nil {
		return err
	}
	current := flags["notify-keyspace-events"]
	if strings.Contains(current, "K") {
		return nil
	}
	updated := current + "K"
	if current == "" {
		updated = "KA"
	}
	return c.rdb.ConfigSet(ctx, "notify-keyspace-events", updated).Err()
}
