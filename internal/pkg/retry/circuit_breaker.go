package retry

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type CircuitBreaker struct {
	mu                  sync.Mutex
	state               State
	consecutiveFailures int
	maxFailures         int
	cooldown            time.Duration
	halfOpenMax         int
	halfOpenCount       int
	lastFailureTime     time.Time
}

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func NewCircuitBreaker(maxFailures int, cooldown time.Duration, halfOpenMax int) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		cooldown:    cooldown,
		halfOpenMax: halfOpenMax,
		state:       StateClosed,
	}
}

func (cb *CircuitBreaker) Execute(operation func() error) error {
	cb.mu.Lock()
	switch cb.state {
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.cooldown {
			cb.state = StateHalfOpen
			cb.halfOpenCount = 0
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	case StateHalfOpen:
		if cb.halfOpenCount >= cb.halfOpenMax {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
		cb.halfOpenCount++
	}
	cb.mu.Unlock()

	err := operation()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.consecutiveFailures++
		cb.lastFailureTime = time.Now()
		if cb.consecutiveFailures >= cb.maxFailures {
			cb.state = StateOpen
		}
		return err
	}

	cb.consecutiveFailures = 0
	cb.state = StateClosed
	return nil
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state == StateOpen
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.consecutiveFailures = 0
	cb.halfOpenCount = 0
}
