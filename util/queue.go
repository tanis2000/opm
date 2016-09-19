package util

import (
	"errors"
	"time"
)

type TrainerQueue struct {
	in     chan *TrainerSession
	out    chan *TrainerSession
	buffer []*TrainerSession
}

// NewTrainerQueue creates a new buffered queue of *TrainerSessions.
// The initial buffer is filled with the provided *TrainerSessions.
func NewTrainerQueue(trainers []*TrainerSession) *TrainerQueue {
	// Create queue
	tq := &TrainerQueue{
		in:     make(chan *TrainerSession),
		out:    make(chan *TrainerSession),
		buffer: trainers,
	}
	// Start *TrainerSession queue/dequeue
	go tq.queue()
	// Return TrainerQueue
	return tq
}

// queue handles the buffer and sends/receives trainers on in/out channels
func (t *TrainerQueue) queue() {
	for {
		if len(t.buffer) > 0 {
			select {
			case t.out <- t.buffer[0]:
				t.buffer = t.buffer[1:]
			case s := <-t.in:
				t.buffer = append(t.buffer, s)
			}
		} else {
			s := <-t.in
			t.buffer = append(t.buffer, s)
		}
	}
}

// Get requests a *TrainerSession from the queue
// This will block until a *TrainerSession is available
func (t *TrainerQueue) Get(timeout time.Duration) (*TrainerSession, error) {
	select {
	case trainer := <-t.out:
		return trainer, nil
	case <-time.After(timeout):
		return &TrainerSession{}, errors.New("Timeout")
	}
}

// Queue returns a *TrainerSession to the queue. Also adds new *TrainerSessions.
func (t *TrainerQueue) Queue(ts *TrainerSession, delay time.Duration) {
	if ts.Account.Banned || ts.Proxy.Dead {
		return
	}
	go func(x *TrainerSession) {
		time.Sleep(delay)
		t.in <- x
	}(ts)
}
