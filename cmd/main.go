package main

import (
	"fmt"
	"netmonitor/pkg/monitor"
	"time"
)

const checkInterval = 5 * time.Second

func main() {
	fmt.Println("Starting port monitoring...")

	listenerMon := monitor.NewListenerMonitor()
	if err := listenerMon.Initialize(); err != nil {
		panic(err)
	}

	establishedMon := monitor.NewEstablishedMonitor()
	if err := establishedMon.Initialize(); err != nil {
		panic(err)
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		// Check listener changes
		newListeners, err := listenerMon.CheckChanges()
		if err != nil {
			fmt.Printf("Error checking listeners: %v\n", err)
			continue
		}
		if len(newListeners) > 0 {
			listenerMon.LogNewListeners(newListeners)
		}

		// Check established connections
		newEstablished, err := establishedMon.CheckChanges()
		if err != nil {
			fmt.Printf("Error checking established connections: %v\n", err)
			continue
		}
		if len(newEstablished) > 0 {
			establishedMon.LogNewConnections(newEstablished)
		}
	}
}