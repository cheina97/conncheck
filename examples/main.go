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
		if err := cc.AddAndRunSender("ID1", "127.0.0.1", updateCallback); err != nil {
			klog.Exitf("failed to add and run sender: %v", err)
		}
	}()

	sw.Wait()
}
