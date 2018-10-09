package botmaid

import (
	"time"
)

// Timer is a func with time and frequency so that we can call it at some
// specific time.
type Timer struct {
	Do        func()
	Time      time.Time
	Frequency string
}

// AddTimer adds a timer into the []Timer.
func AddTimer(ts *[]Timer, f func(), t time.Time, fre string) {
	*ts = append(*ts, Timer{
		Do:        f,
		Time:      t,
		Frequency: fre,
	})
}
