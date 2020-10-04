package timercheck

import (
	"sync"
	"time"
)

/**
	TimerCheck is a simple throttler, can limit the call frequency of
a single function. The difference between TimerCheck and Throttler is
that TimerCheck can let all the request before a action take effect,
but Throttler let some request after a action take effect. At the same
time TimerCheck can do something immediately and do others periodicity
through one lock.
*/
type TimerCheck struct {
	needUpdate bool
	mux        sync.Mutex
	circle     time.Duration
	callback   func()
}

func MakeTimerChecker(circle time.Duration, callback func()) *TimerCheck {
	return &TimerCheck{
		needUpdate: false,
		circle:     circle,
		callback:   callback,
	}
}

// 定时器，周期检查是否要更新配置的需求
func (timer *TimerCheck) DoCircle() {
	for range time.Tick(timer.circle) {
		timer.mux.Lock()
		if timer.needUpdate {
			timer.callback()
			timer.needUpdate = false
		}
		timer.mux.Unlock()
	}
}

// get lock do something, and active a task circle
func (timer *TimerCheck) Update(task func()) {
	timer.mux.Lock()
	task()
	timer.needUpdate = true
	timer.mux.Unlock()
}
