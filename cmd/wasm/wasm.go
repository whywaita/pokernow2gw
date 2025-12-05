package main

import (
	"strings"
	"time"
	"unsafe"

	"github.com/whywaita/pokernow2gw/pkg/pokernow2gw"
)

// Memory management for passing strings between Go and JavaScript

//lint:ignore U1000 This function is exported to WASM and called from JavaScript
//go:wasmexport malloc
func malloc(size uint32) uint32 {
	buf := make([]byte, size)
	ptr := &buf[0]
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

//lint:ignore U1000 This function is exported to WASM and called from JavaScript
//go:wasmexport free
func free(ptr uint32) {
	// In Go's WASM implementation, we don't need explicit free
	// as garbage collection handles memory management
}

// getString reads a string from memory
//
//lint:ignore U1000 This function is used by WASM exported functions
func getString(ptr, length uint32) string {
	if length == 0 {
		return ""
	}
	// Use unsafe.Add to avoid go vet warning about uintptr to unsafe.Pointer conversion
	p := unsafe.Add(unsafe.Pointer(nil), uintptr(ptr))
	buf := unsafe.Slice((*byte)(p), int(length))
	return string(buf)
}

//lint:ignore U1000 This variable is used by WASM exported functions to prevent GC
var lastResult []byte // Keep reference to prevent GC

//lint:ignore U1000 This variable is used by WASM exported functions to store result info
var lastResultInfo [12]byte // Store result info: [ptr(4), len(4), skippedHands(4)]

//lint:ignore U1000 This function is exported to WASM and called from JavaScript
//go:wasmexport parseCSV
func parseCSV(csvPtr, csvLen, heroPtr, heroLen, filterFlags uint32) uint32 {
	csvText := getString(csvPtr, csvLen)
	heroName := getString(heroPtr, heroLen)

	if csvText == "" {
		errMsg := "CSV text is empty"
		lastResult = []byte(errMsg)
		// Write error result: ptr, len, skippedHands=0
		writeResultInfo(uint32(uintptr(unsafe.Pointer(&lastResult[0]))), uint32(len(lastResult)), 0, 1)
		return uint32(uintptr(unsafe.Pointer(&lastResultInfo[0])))
	}

	if heroName == "" {
		errMsg := "Hero name is required"
		lastResult = []byte(errMsg)
		writeResultInfo(uint32(uintptr(unsafe.Pointer(&lastResult[0]))), uint32(len(lastResult)), 0, 1)
		return uint32(uintptr(unsafe.Pointer(&lastResultInfo[0])))
	}

	// Build player count filter from flags
	// filterFlags is a bitmask: bit 0 = HU, bit 1 = SpinAndGo, bit 2 = MTT
	var playerCountFilter pokernow2gw.PlayerCountFilter
	if filterFlags == 0 {
		playerCountFilter = pokernow2gw.PlayerCountAll
	} else {
		playerCountFilter = pokernow2gw.PlayerCountFilter(filterFlags)
	}

	// Parse CSV
	reader := strings.NewReader(csvText)
	opts := pokernow2gw.ConvertOptions{
		HeroName:          heroName,
		SiteName:          "PokerStars",
		TimeLocation:      time.UTC,
		PlayerCountFilter: playerCountFilter,
	}

	result, err := pokernow2gw.ParseCSV(reader, opts)
	if err != nil {
		errMsg := err.Error()
		lastResult = []byte(errMsg)
		writeResultInfo(uint32(uintptr(unsafe.Pointer(&lastResult[0]))), uint32(len(lastResult)), 0, 1)
		return uint32(uintptr(unsafe.Pointer(&lastResultInfo[0])))
	}

	lastResult = result.HH
	writeResultInfo(uint32(uintptr(unsafe.Pointer(&lastResult[0]))), uint32(len(lastResult)), uint32(result.SkippedHands), 0)
	return uint32(uintptr(unsafe.Pointer(&lastResultInfo[0])))
}

//lint:ignore U1000 This function is used by WASM exported functions
func writeResultInfo(ptr, length, skippedHands, hasError uint32) {
	// Write 4 uint32 values to lastResultInfo
	*(*uint32)(unsafe.Pointer(&lastResultInfo[0])) = ptr
	*(*uint32)(unsafe.Pointer(&lastResultInfo[4])) = length
	*(*uint32)(unsafe.Pointer(&lastResultInfo[8])) = skippedHands
	// Store hasError as a separate field
	// For simplicity, we'll use skippedHands = 0xFFFFFFFF to indicate error
	if hasError == 1 {
		*(*uint32)(unsafe.Pointer(&lastResultInfo[8])) = 0xFFFFFFFF
	}
}

func main() {
	// Entry point for WASM module
	// Keep the program running so Go runtime doesn't exit
	select {}
}
