package util

import "time"

type ExponentialBackoffParameters struct {
	InitialSleepDuration time.Duration
	DurationMultiplier   int
	MaxSleepDuration     time.Duration
}

// Starts a new goroutine which executes a task repeatedly with exponential backoff.
//
// The task can return true to stop the routine, else false to continue.
// Can specify an initial sleep duration, the multiplier to use, and a max sleep duration.
func StartExponentialBackoffRoutine(task func() bool, params ExponentialBackoffParameters) {
	go func() {
		sleepDuration := params.InitialSleepDuration

		for {
			time.Sleep(sleepDuration)

			if task() {
				break
			}

			sleepDuration *= time.Duration(params.DurationMultiplier)
			if sleepDuration > params.MaxSleepDuration {
				sleepDuration = params.MaxSleepDuration
			}
		}
	}()
}
