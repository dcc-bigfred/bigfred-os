package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const DefaultAddr = "127.0.0.1:6379"

var ErrNotFound = errors.New("redis key not found")

// KeySummary is a key name with its TTL in seconds (-1 = no expiry).
type KeySummary struct {
	Key string `json:"key"`
	TTL int64  `json:"ttl"`
}

// KeyDetail is the value and metadata for a single key.
type KeyDetail struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	TTL   int64  `json:"ttl"`
	Value any    `json:"value"`
}

// Client wraps a go-redis connection to localhost Redis.
type Client struct {
	addr string
	rdb  *goredis.Client
}

// NewClient dials addr (default 127.0.0.1:6379).
func NewClient(addr string) *Client {
	if addr == "" {
		addr = DefaultAddr
	}
	return &Client{
		addr: addr,
		rdb:  goredis.NewClient(&goredis.Options{Addr: addr}),
	}
}

// Ping verifies the server is reachable.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// ListKeys scans keys matching pattern and returns each key with TTL.
func (c *Client) ListKeys(ctx context.Context, pattern string) ([]KeySummary, error) {
	if pattern == "" {
		pattern = "*"
	}
	var out []KeySummary
	iter := c.rdb.Scan(ctx, 0, pattern, 200).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		ttl, err := c.rdb.TTL(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		out = append(out, KeySummary{Key: key, TTL: ttlSeconds(ttl)})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []KeySummary{}
	}
	return out, nil
}

// GetKey returns type, TTL, and a JSON-friendly value for key.
func (c *Client) GetKey(ctx context.Context, key string) (KeyDetail, error) {
	if key == "" {
		return KeyDetail{}, fmt.Errorf("empty key")
	}
	exists, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return KeyDetail{}, err
	}
	if exists == 0 {
		return KeyDetail{}, ErrNotFound
	}

	ttl, err := c.rdb.TTL(ctx, key).Result()
	if err != nil {
		return KeyDetail{}, err
	}
	keyType, err := c.rdb.Type(ctx, key).Result()
	if err != nil {
		return KeyDetail{}, err
	}

	value, err := readValue(ctx, c.rdb, key, keyType)
	if err != nil {
		return KeyDetail{}, err
	}
	return KeyDetail{
		Key:   key,
		Type:  keyType,
		TTL:   ttlSeconds(ttl),
		Value: value,
	}, nil
}

// DeleteKey removes key from Redis.
func (c *Client) DeleteKey(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("empty key")
	}
	n, err := c.rdb.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func ttlSeconds(d time.Duration) int64 {
	return int64(d / time.Second)
}

func readValue(ctx context.Context, rdb *goredis.Client, key, keyType string) (any, error) {
	switch keyType {
	case "string":
		return rdb.Get(ctx, key).Result()
	case "hash":
		return rdb.HGetAll(ctx, key).Result()
	case "list":
		return rdb.LRange(ctx, key, 0, -1).Result()
	case "set":
		return rdb.SMembers(ctx, key).Result()
	case "zset":
		return rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
	case "stream":
		entries, err := rdb.XRange(ctx, key, "-", "+").Result()
		if err != nil {
			return nil, err
		}
		return entries, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", keyType)
	}
}

// Addr returns the configured Redis address.
func (c *Client) Addr() string { return c.addr }
