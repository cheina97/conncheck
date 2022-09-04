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

package main

import (
	"flag"
	"sync"

	"github.com/cheina97/conncheck"
	"k8s.io/klog/v2"
)

func updateCallback(connected bool) error {
	klog.Infof("Connected: %v", connected)
	return nil
}

func main() {
	target := flag.String("target", "", "target ip")
	klog.InitFlags(nil)
	flag.Parse()

	sw := sync.WaitGroup{}
	sw.Add(1)

	var cc *conncheck.ConnChecker
	var err error
	if cc, err = conncheck.NewConnChecker(); err != nil {
		klog.Exitf("failed to create connchecker: %v", err)
	}

	go func() {
		if err := cc.RunReceiver(); err != nil {
			klog.Exitf("failed to run receiver: %v", err)
		}
	}()

	go func() {
		if err := cc.RunReceiverDisconnectObserver(); err != nil {
			klog.Exitf("failed to run receiver disconnect checker: %v", err)
		}
	}()

	go func() {
		if err := cc.AddAndRunSender("ID1", *target, updateCallback); err != nil {
			klog.Exitf("failed to add and run sender: %v", err)
		}
	}()

	/* go func() {
		wait.Forever(func() {
			latency, err := cc.GetLatency("ID1")
			if err != nil {
				klog.Errorf("failed to get latency: %v", err)
			}
			klog.Infof("latency: %v", latency)
		}, time.Second)
	}() */

	sw.Wait()
}
