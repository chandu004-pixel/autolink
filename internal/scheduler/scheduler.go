package scheduler

import (
	"github.com/user/autolink/internal/logging"
)

type Limiter struct {
	MaxActions int
	Count      int
}

func New(max int) *Limiter {
	return &Limiter{MaxActions: max}
}

func (l *Limiter) ShouldWait() bool {
	if l.Count >= l.MaxActions {
		logging.Logger.Warn("Daily limit reached, slowing down...")
		return true
	}
	return false
}

func (l *Limiter) Increment() {
	l.Count++
}

func (l *Limiter) Reset() {
	l.Count = 0
}
