package conncheck

import "time"

const (
	port          = 12345
	buffSize      = 1024
	NoneClusterID = "None"
)

var (
	// Timeout is the timeout for the conncheck ping in seconds.
	Timeout = 10 * time.Second
	// PeriodicPingInterval is the interval for the periodic ping in seconds.
	PeriodicPingInterval = 2 * time.Second
	// MaxConsecutiveErrors is the maximum number of consecutive errors.
	MaxConsecutiveErrors = 5
)
