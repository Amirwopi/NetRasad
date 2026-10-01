package main

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	iphlpapi         = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetIfTable2  = iphlpapi.NewProc("GetIfTable2")
	procFreeMibTable = iphlpapi.NewProc("FreeMibTable")
)

func isFilterDriver(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "filter") ||
		strings.Contains(lower, "qos") ||
		strings.Contains(lower, "wfp") ||
		strings.Contains(lower, "winpk") ||
		strings.Contains(lower, "packet scheduler")
}

func main() {
	fmt.Println("==================================================")
	fmt.Println(" Real-Time Counter Verification (Offsets 1280 & 1320)")
	fmt.Println("==================================================")

	prevCounters := make(map[uint64]struct {
		InOctets  uint64
		OutOctets uint64
		Time      time.Time
	})

	for poll := 1; poll <= 6; poll++ {
		var tablePtr unsafe.Pointer
		r1, _, _ := procGetIfTable2.Call(uintptr(unsafe.Pointer(&tablePtr)))
		if r1 != 0 || tablePtr == nil {
			fmt.Printf("GetIfTable2 failed: %d\n", r1)
			time.Sleep(1 * time.Second)
			continue
		}

		numEntries := *(*uint32)(tablePtr)
		rowPtr := unsafe.Add(tablePtr, 8)
		rowBytes := 1352
		now := time.Now()

		fmt.Printf("\n--- Poll #%d [%s] (Total Rows: %d) ---\n", poll, now.Format("15:04:05"), numEntries)

		for i := uint32(0); i < numEntries; i++ {
			currRow := unsafe.Add(rowPtr, int(i)*rowBytes)
			luid := *(*uint64)(currRow)
			aliasPtr := (*[257]uint16)(unsafe.Add(currRow, 28))
			alias := windows.UTF16ToString(aliasPtr[:])

			// Skip NDIS filter driver duplicate sub-interfaces
			if isFilterDriver(alias) {
				continue
			}

			// Read InOctets at offset 1280 and OutOctets at offset 1320
			inOctets := *(*uint64)(unsafe.Add(currRow, 1280))
			outOctets := *(*uint64)(unsafe.Add(currRow, 1320))

			prev, found := prevCounters[luid]
			prevCounters[luid] = struct {
				InOctets  uint64
				OutOctets uint64
				Time      time.Time
			}{
				InOctets:  inOctets,
				OutOctets: outOctets,
				Time:      now,
			}

			if !found {
				if inOctets > 0 || outOctets > 0 {
					fmt.Printf("  [Init] LUID: 0x%016x | Name: %-25q | In: %d bytes | Out: %d bytes\n",
						luid, alias, inOctets, outOctets)
				}
				continue
			}

			elapsed := now.Sub(prev.Time).Seconds()
			if elapsed <= 0 {
				elapsed = 1.0
			}

			downDelta := int64(inOctets) - int64(prev.InOctets)
			upDelta := int64(outOctets) - int64(prev.OutOctets)
			if downDelta < 0 {
				downDelta = int64(inOctets)
			}
			if upDelta < 0 {
				upDelta = int64(outOctets)
			}

			downBps := float64(downDelta) / elapsed
			upBps := float64(upDelta) / elapsed
			downKBs := downBps / 1024
			upKBs := upBps / 1024
			downMbit := (downBps * 8) / 1_000_000
			upMbit := (upBps * 8) / 1_000_000

			if inOctets > 0 || outOctets > 0 {
				fmt.Printf("  [LIVE] Name: %-25q | Down: %7.2f KB/s (%5.2f Mbit) | Up: %7.2f KB/s (%5.2f Mbit) | TotalDown: %d | TotalUp: %d\n",
					alias, downKBs, downMbit, upKBs, upMbit, inOctets, outOctets)
			}
		}

		procFreeMibTable.Call(uintptr(tablePtr))
		time.Sleep(1 * time.Second)
	}
}
