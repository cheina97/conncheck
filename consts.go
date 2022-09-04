// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conncheck

import "time"

const (
	port     = 12345
	buffSize = 1024
	// ConnectionErrorMessage is the message sent when a connection error occurs.
	ConnectionErrorMessage = "Cannot ping the other cluster."
)

var (
	// ConnectionCheckInterval is the interval at which the connection check is performed.
	ConnectionCheckInterval time.Duration = 1 * time.Second
	// ExceedingTime is the time after which the connection check is considered as failed.
	ExceedingTime time.Duration = 5 * time.Second
	// PeriodicPingInterval is the interval at which the ping is sent.
	PeriodicPingInterval time.Duration = 1 * time.Second
)
