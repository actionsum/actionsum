package main

import (
	"fmt"
	"log"
	"time"

	"actionsum/pkg/detector"
)

func main() {
	fmt.Println("Testing IntegrationsV2 Hybrid Detector")
	fmt.Println("======================================")

	// Create V2 detector
	det, err := detector.NewV2()
	if err != nil {
		log.Fatalf("Failed to create detector: %v", err)
	}
	defer det.Close()

	fmt.Printf("\nDisplay Server: %s\n", det.GetDisplayServer())
	fmt.Printf("Is Available: %v\n\n", det.IsAvailable())

	// Test detection for 30 seconds
	fmt.Println("Monitoring active window for 30 seconds...")
	fmt.Println("Switch between different applications to test detection")
	fmt.Println()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)
	count := 0

	for {
		select {
		case <-timeout:
			fmt.Println("\nTest completed!")
			return

		case <-ticker.C:
			count++
			windowInfo, err := det.GetFocusedWindow()
			if err != nil {
				log.Printf("[%d] Error: %v", count, err)
				continue
			}

			if windowInfo == nil {
				log.Printf("[%d] No window detected", count)
				continue
			}

			fmt.Printf("[%d] App: %-20s | Title: %-50s | Display: %s\n",
				count,
				truncate(windowInfo.AppName, 20),
				truncate(windowInfo.WindowTitle, 50),
				windowInfo.DisplayServer,
			)

			// Check idle status
			idleInfo, err := det.GetIdleInfo()
			if err == nil && idleInfo != nil {
				if idleInfo.IsIdle || idleInfo.IsLocked {
					fmt.Printf("     System: Idle=%v, Locked=%v\n", idleInfo.IsIdle, idleInfo.IsLocked)
				}
			}
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
