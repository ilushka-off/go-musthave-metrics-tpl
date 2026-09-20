// Package retry provides a small generic retry helper with fixed,
// caller-supplied backoff delays.
package retry

import "time"

// Delays is the default backoff schedule used for retrying storage
// operations: three attempts spaced 1s, 3s, and 5s apart.
var Delays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

// Do runs op, retrying it after each delay in delays as long as the
// previous error is retriable according to isRetriable. It returns nil as
// soon as op succeeds, or the last error once delays is exhausted or
// isRetriable reports false.
func Do(delays []time.Duration, isRetriable func(error) bool, op func() error) error {
	err := op()
	if err == nil {
		return nil
	}

	for _, delay := range delays {
		if !isRetriable(err) {
			return err
		}
		time.Sleep(delay)
		err = op()
		if err == nil {
			return nil
		}
	}
	return err
}
