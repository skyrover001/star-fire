package main

import (
	"errors"
	"testing"
	"time"
)

func TestReconnectBackoffSequenceCapAndReset(t *testing.T) {
	backoff := newReconnectBackoff()
	want := []time.Duration{3, 6, 12, 24, 48, 60, 60}
	for index, seconds := range want {
		if got := backoff.Next(); got != seconds*time.Second {
			t.Fatalf("delay %d: got %v, want %v", index, got, seconds*time.Second)
		}
	}

	backoff.Reset()
	if got := backoff.Next(); got != initialReconnectDelay {
		t.Fatalf("delay after reset: got %v, want %v", got, initialReconnectDelay)
	}
}

func TestPermanentRegistrationErrors(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		permanent bool
	}{
		{name: "unauthorized", err: errors.New("bad status: 401 Unauthorized"), permanent: true},
		{name: "forbidden", err: errors.New("bad status: 403 Forbidden"), permanent: true},
		{name: "connection refused", err: errors.New("actively refused it"), permanent: false},
		{name: "temporary server error", err: errors.New("bad status: 503 Service Unavailable"), permanent: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isPermanentRegistrationError(test.err); got != test.permanent {
				t.Fatalf("got %v, want %v", got, test.permanent)
			}
		})
	}
}

func TestWaitForReconnectStopsOnQuit(t *testing.T) {
	quit := make(chan struct{})
	close(quit)
	if waitForReconnect(quit, time.Hour) {
		t.Fatal("expected closed quit channel to cancel retry wait")
	}
}
