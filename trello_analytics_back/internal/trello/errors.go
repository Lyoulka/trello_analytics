package trello

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
)

// Sentinel errors for common Trello HTTP statuses (use with errors.Is).
var (
	ErrUnauthorized = errors.New("trello: unauthorized")
	ErrForbidden    = errors.New("trello: forbidden")
	ErrNotFound     = errors.New("trello: not found")
	ErrRateLimited  = errors.New("trello: rate limited")
)

// IsUnauthorized reports whether err is a 401 from Trello.
func IsUnauthorized(err error) bool { return errors.Is(err, ErrUnauthorized) }

// IsForbidden reports whether err is a 403 from Trello.
func IsForbidden(err error) bool { return errors.Is(err, ErrForbidden) }

// IsNotFound reports whether err is a 404 from Trello.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// IsRateLimited reports whether err is a 429 from Trello.
func IsRateLimited(err error) bool { return errors.Is(err, ErrRateLimited) }

// mapAPIError attaches sentinel causes for 401/403/404/429 so callers can classify
// without parsing status codes.
func mapAPIError(e *APIError) error {
	if e == nil {
		return nil
	}
	switch e.StatusCode {
	case 401:
		return fmt.Errorf("%w: %w", ErrUnauthorized, e)
	case 403:
		return fmt.Errorf("%w: %w", ErrForbidden, e)
	case 404:
		return fmt.Errorf("%w: %w", ErrNotFound, e)
	case 429:
		return fmt.Errorf("%w: %w", ErrRateLimited, e)
	default:
		return e
	}
}

// IsRetryable reports whether a Trello client error should be retried per policy
// (see retry.go). Non-retryable: 401/403/404 and context cancellation.
//
// Retryable: 429, 500, 502, 503, 504, network/timeout transport errors.
//
// Note: live 429 was not_reproduced in the history spike — treat 429 as retryable
// by HTTP semantics only; do not claim a confirmed Trello rate-limit profile.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
		return false
	}
	if errors.Is(err, ErrRateLimited) {
		return true
	}
	var api *APIError
	if errors.As(err, &api) {
		switch api.StatusCode {
		case 429, 500, 502, 503, 504:
			return true
		case 401, 403, 404:
			return false
		default:
			if api.StatusCode >= 400 && api.StatusCode < 500 {
				return false
			}
			if api.StatusCode >= 500 {
				return true
			}
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	// Client timeout often wraps as context.DeadlineExceeded without net.Error.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return false
}

// RetryAfterDuration extracts Retry-After from a wrapped APIError, if any.
func RetryAfterDuration(err error) time.Duration {
	var api *APIError
	if errors.As(err, &api) && api != nil {
		return api.RetryAfter
	}
	return 0
}
