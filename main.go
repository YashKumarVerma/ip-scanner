package main

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Port Scanner")

	// UI elements
	startIPEntry := widget.NewEntry()
	startIPEntry.SetPlaceHolder("Start IP")
	endIPEntry := widget.NewEntry()
	endIPEntry.SetPlaceHolder("End IP")
	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("Port")
	timeoutEntry := widget.NewEntry()
	timeoutEntry.SetPlaceHolder("Timeout (ms)")
	workersEntry := widget.NewEntry()
	workersEntry.SetPlaceHolder("Workers")

	progress := widget.NewProgressBar()
	results := widget.NewMultiLineEntry()
	results.SetPlaceHolder("Results will be displayed here...")

	scanButton := widget.NewButton("Scan", func() {
		startIP := startIPEntry.Text
		endIP := endIPEntry.Text
		port := portEntry.Text
		timeout := timeoutEntry.Text
		workers := workersEntry.Text

		progress.SetValue(0)
		results.SetText("")

		if startIP == "" || endIP == "" || port == "" || timeout == "" || workers == "" {
			results.SetText("Please fill in all fields")
			return
		}

		go func() {
			networkScan(startIP, endIP, port, timeout, workers, progress, results)
		}()
	})

	// Layout
	myWindow.SetContent(container.NewVBox(
		widget.NewLabel("Port Scanner"),
		startIPEntry,
		endIPEntry,
		portEntry,
		timeoutEntry,
		workersEntry,
		progress,
		results,
		scanButton,
	))

	myWindow.Resize(fyne.NewSize(600, 400))
	myWindow.ShowAndRun()
}

func networkScan(startIP, endIP, port, timeout, workers string, progress *widget.ProgressBar, results *widget.Entry) {
	start := ipToUint32(net.ParseIP(startIP))
	end := ipToUint32(net.ParseIP(endIP))
	workerCount, _ := strconv.Atoi(workers)
	timeoutMs, _ := strconv.Atoi(timeout)

	ipChan := make(chan uint32, workerCount)
	resultChan := make(chan string, workerCount)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		go func() {
			for ip := range ipChan {
				address := fmt.Sprintf("%s:%s", uint32ToIP(ip).String(), port)
				if isPortOpen(address, timeoutMs) {
					mu.Lock()
					resultChan <- address
					mu.Unlock()
				}
				wg.Done()
			}
		}()
	}

	// Close result channel when done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	totalIPs := float64(end - start + 1)
	current := 0.0

	// Enqueue IPs
	for ip := start; ip <= end; ip++ {
		wg.Add(1)
		ipChan <- ip
		mu.Lock()
		current++
		progress.SetValue(current / totalIPs)
		mu.Unlock()
	}
	close(ipChan)

	// Collect and display results
	for openAddr := range resultChan {
		mu.Lock()
		results.SetText(results.Text + openAddr + "\n")
		mu.Unlock()
	}
	progress.SetValue(1)
}

func isPortOpen(address string, timeoutMs int) bool {
	conn, err := net.DialTimeout("tcp", address, time.Duration(timeoutMs)*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func ipToUint32(ip net.IP) uint32 {
	ipv4 := ip.To4()
	return uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])
}

func uint32ToIP(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}
