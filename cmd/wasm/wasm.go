package main

import (
	"encoding/json"
	"strings"
	"time"
	"unsafe"

	"github.com/whywaita/pokernow2gw/pkg/pokernow2gw"
)

const (
	// wasmErrorSentinel is written to the skippedHands field of the result info
	// to signal an error to JavaScript. When JavaScript reads this value from
	// the skippedHands position, it should treat the result as an error and
	// interpret the ptr/len fields as an error message instead of HH output.
	wasmErrorSentinel = 0xFFFFFFFF
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

//lint:ignore U1000 This variable is used by WASM exported functions to store skipped hands detail JSON
var lastSkippedDetail []byte // Keep reference to prevent GC

//lint:ignore U1000 This variable is used by WASM exported functions to store result info
var lastResultInfo [20]byte // Store result info: [ptr(4), len(4), skippedHands(4), skippedDetailPtr(4), skippedDetailLen(4)]

//lint:ignore U1000 This function is exported to WASM and called from JavaScript
//go:wasmexport parseCSV
func parseCSV(csvPtr, csvLen, heroPtr, heroLen, filterFlags uint32, rakePercent, rakeCapBB float32) uint32 {
	csvText := getString(csvPtr, csvLen)
	heroName := getString(heroPtr, heroLen)

	if csvText == "" {
		errMsg := "CSV text is empty"
		lastResult = []byte(errMsg)
		lastSkippedDetail = nil
		// Write error result: ptr, len, skippedHands=0
		writeResultInfo(uint32(uintptr(unsafe.Pointer(&lastResult[0]))), uint32(len(lastResult)), 0, 1, 0, 0)
		return uint32(uintptr(unsafe.Pointer(&lastResultInfo[0])))
	}

	if heroName == "" {
		errMsg := "Hero name is required"
		lastResult = []byte(errMsg)
		lastSkippedDetail = nil
		writeResultInfo(uint32(uintptr(unsafe.Pointer(&lastResult[0]))), uint32(len(lastResult)), 0, 1, 0, 0)
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
		RakePercent:       float64(rakePercent),
		RakeCapBB:         float64(rakeCapBB),
	}

	result, err := pokernow2gw.Parse(reader, opts)
	if err != nil {
		errMsg := err.Error()
		lastResult = []byte(errMsg)
		lastSkippedDetail = nil
		writeResultInfo(uint32(uintptr(unsafe.Pointer(&lastResult[0]))), uint32(len(lastResult)), 0, 1, 0, 0)
		return uint32(uintptr(unsafe.Pointer(&lastResultInfo[0])))
	}

	lastResult = result.HH

	// Encode skipped hands info as JSON
	var skippedDetailPtr, skippedDetailLen uint32
	if len(result.SkippedHandsInfo) > 0 {
		jsonData, err := json.Marshal(result.SkippedHandsInfo)
		if err == nil {
			lastSkippedDetail = jsonData
			skippedDetailPtr = uint32(uintptr(unsafe.Pointer(&lastSkippedDetail[0])))
			skippedDetailLen = uint32(len(lastSkippedDetail))
		}
	} else {
		lastSkippedDetail = nil
	}

	writeResultInfo(uint32(uintptr(unsafe.Pointer(&lastResult[0]))), uint32(len(lastResult)), uint32(result.SkippedHands), 0, skippedDetailPtr, skippedDetailLen)
	return uint32(uintptr(unsafe.Pointer(&lastResultInfo[0])))
}

//lint:ignore U1000 This function is used by WASM exported functions
func writeResultInfo(ptr, length, skippedHands, hasError, skippedDetailPtr, skippedDetailLen uint32) {
	// Write 5 uint32 values to lastResultInfo
	*(*uint32)(unsafe.Pointer(&lastResultInfo[0])) = ptr
	*(*uint32)(unsafe.Pointer(&lastResultInfo[4])) = length
	*(*uint32)(unsafe.Pointer(&lastResultInfo[8])) = skippedHands
	*(*uint32)(unsafe.Pointer(&lastResultInfo[12])) = skippedDetailPtr
	*(*uint32)(unsafe.Pointer(&lastResultInfo[16])) = skippedDetailLen
	// When hasError is set, overwrite skippedHands with the error sentinel value.
	// JavaScript checks for this sentinel to distinguish errors from normal results.
	if hasError == 1 {
		*(*uint32)(unsafe.Pointer(&lastResultInfo[8])) = wasmErrorSentinel
	}
}

func main() {
	// Entry point for WASM module
	// Keep the program running so Go runtime doesn't exit
	select {}
}
