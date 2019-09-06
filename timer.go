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
func (bm *BotMaid) AddTimer(t Timer) {
	bm.Timers = append(bm.Timers, t)
}

func (bm *BotMaid) loadTimers() {
	for i := range bm.Timers {
		v := bm.Timers[i]
		next := v.Start

		if v.Frequency == 0 && time.Now().After(next) {
			continue
		}

		go func(v *Timer) {
			for {
				for time.Now().After(next) {
					next = next.Add(v.Frequency)
				}

				if !v.End.IsZero() && next.After(v.End) {
					break
				}

				timer := time.NewTimer(-time.Since(next))
				<-timer.C
				v.Do()

				if v.Frequency == 0 {
					break
				}
			}
		}(&bm.Timers[i])
	}
}
