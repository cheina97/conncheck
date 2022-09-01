package conncheck

import "time"

const (
	port          = 12345
	buffSize      = 1024
	NoneClusterID = "None"
)

var (
	// Timeout is the timeout for the conncheck ping in seconds.
	Timeout = 1 * time.Second
	// PeriodicPingInterval is the interval for the periodic ping in seconds.
	PeriodicPingInterval = 1 * time.Second
	// MaxConsecutiveErrors is the maximum number of consecutive errors.
	MaxConsecutiveErrors = 5
	// CleanTime is the time to wait for a msg to be discarded.
	CleanTime = 5 * time.Second
)
