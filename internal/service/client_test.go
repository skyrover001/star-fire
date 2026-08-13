package service

import "testing"

func TestNormalizeUnixMillis(t *testing.T) {
	tests := []struct {
		name      string
		timestamp int64
		want      int64
	}{
		{name: "seconds", timestamp: 1_786_586_146, want: 1_786_586_146_000},
		{name: "milliseconds", timestamp: 1_786_586_146_123, want: 1_786_586_146_123},
		{name: "zero", timestamp: 0, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := normalizeUnixMillis(test.timestamp); got != test.want {
				t.Fatalf("normalizeUnixMillis(%d) = %d, want %d", test.timestamp, got, test.want)
			}
		})
	}
}
