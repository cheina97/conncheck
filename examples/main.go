package main

import (
	"flag"
	"sync"

	"github.com/cheina97/conncheck"
	"k8s.io/klog/v2"
)

func updateCallback(connected bool) error {
	klog.Infof("status callback called: %v", connected)
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
