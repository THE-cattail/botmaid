package botmaid

import (
	"time"
)

// Timer is a func with time and frequency so that we can call it at some
// specific time.
type Timer struct {
	Do         func()
	Start, End time.Time
	Frequency  time.Duration
}

// AddTimer adds a timer into the []Timer.
func (bm *BotMaid) AddTimer(t *Timer) {
	bm.Timers = append(bm.Timers, t)
}

func (bm *BotMaid) loadTimers() {
	for _, t := range bm.Timers {
		tm := t
		next := tm.Start

		if tm.Frequency == 0 && time.Now().After(next) {
			continue
		}

		go func(t *Timer) {
			for {
				for time.Now().After(next) {
					next = next.Add(t.Frequency)
				}

				if !t.End.IsZero() && next.After(t.End) {
					break
				}

				timer := time.NewTimer(-time.Since(next))
				<-timer.C
				t.Do()

				if t.Frequency == 0 {
					break
				}
			}
		}(tm)
	}
}
