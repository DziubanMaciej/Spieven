package common

import "time"

func TryCallWithTimeouts[T any](function func() (T, error), timeout time.Duration, iterations int) (res T, err error) {
	deadline := time.Now()
	timeoutPerIteration := timeout / time.Duration(iterations)

	for range iterations {
		res, err = function()
		if err == nil {
			break
		} else {
			now := time.Now()
			deadline = deadline.Add(timeoutPerIteration)
			if now.Before(deadline) {
				time.Sleep(deadline.Sub(now))
			}
		}

	}
	return

}
