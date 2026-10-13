package trello

import (
	"context"
	"math/rand"
	"time"
)

// Retry / backoff policy for Trello HTTP JSON calls used by importer/sync.
//
// Policy (Architect / OpenSpec 1.8):
//   - Retryable: 429, 500, 502, 503, 504, network/timeout transport errors
//   - Non-retryable: 401, 403, 404 (fail immediately)
//   - Max attempts: 5 per request
//   - Exponential backoff: base 1s ×2^attempt, cap 60s, full jitter [0, base]
//   - On 429: honor Retry-After (seconds or HTTP-date) when present
//
// Caveat: live 429 was not_reproduced in spike-trello-history — this implements
// HTTP retry semantics only and MUST NOT be described as a confirmed Trello rate limit.
const (
	defaultRetryMaxAttempts = 5
	retryBaseDelay          = 1 * time.Second
	retryMaxDelay           = 60 * time.Second
)

// sleepFunc waits for d or until ctx is done. Injectable for tests.
type sleepFunc func(ctx context.Context, d time.Duration) error

func defaultSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) retryMaxAttempts() int {
	if c != nil && c.RetryMaxAttempts > 0 {
		return c.RetryMaxAttempts
	}
	return defaultRetryMaxAttempts
}

func (c *Client) sleep(ctx context.Context, d time.Duration) error {
	if c != nil && c.Sleep != nil {
		return c.Sleep(ctx, d)
	}
	return defaultSleep(ctx, d)
}

func (c *Client) jitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	if c != nil && c.Jitter != nil {
		return c.Jitter(max)
	}
	// Full jitter: uniform in [0, max].
	return time.Duration(rand.Int63n(int64(max) + 1))
}

// computeRetryDelay returns the wait before the next attempt (attempt is 1-based
// index of the failed try: after 1st failure attempt=1 → base 1s).
func (c *Client) computeRetryDelay(attempt int, err error) time.Duration {
	if ra := RetryAfterDuration(err); ra > 0 {
		return ra
	}
	// base = min(1s * 2^(attempt-1), 60s)
	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 31 {
		shift = 31
	}
	base := retryBaseDelay << uint(shift)
	if base > retryMaxDelay || base <= 0 {
		base = retryMaxDelay
	}
	return c.jitter(base)
}

// withRetry runs op up to retryMaxAttempts times with backoff on retryable errors.
func (c *Client) withRetry(ctx context.Context, op func() error) error {
	max := c.retryMaxAttempts()
	var lastErr error
	for attempt := 1; attempt <= max; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := op()
		if err == nil {
			return nil
		}
		lastErr = err
		if !IsRetryable(err) || attempt == max {
			return err
		}
		delay := c.computeRetryDelay(attempt, err)
		if sleepErr := c.sleep(ctx, delay); sleepErr != nil {
			return sleepErr
		}
	}
	return lastErr
}
