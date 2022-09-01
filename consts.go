package conncheck

import "time"

const (
	port          = 12345
	buffSize      = 1024
	NoneClusterID = "None"
)

var (
	TimeExceededCheckInterval = 5 * time.Second
	ExceedingTime             = 10 * time.Second
	PeriodicPingInterval      = 2 * time.Second
)
