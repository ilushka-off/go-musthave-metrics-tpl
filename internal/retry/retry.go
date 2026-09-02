package retry

import "time"

var Delays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

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
