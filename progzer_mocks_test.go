package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// mockTime is a struct that implements a mock time for testing
type mockTime struct {
	currentTime time.Time
}

func newMockTime(startTime time.Time) *mockTime {
	return &mockTime{
		currentTime: startTime,
	}
}

func (mt *mockTime) Now() time.Time {
	return mt.currentTime
}

func (mt *mockTime) Advance(d time.Duration) {
	mt.currentTime = mt.currentTime.Add(d)
}

// TestUpdateDisplayOutput tests the output of updateDisplay
func TestUpdateDisplayOutput(t *testing.T) {
	// Create a buffer to capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Restore stderr when done
	defer func() {
		os.Stderr = oldStderr
	}()

	// Create a Progress instance
	p := &Progress{
		bytesRead:   500,
		totalSize:   1000,
		startTime:   time.Now().Add(-10 * time.Second),
		lastUpdate:  time.Now().Add(-1 * time.Second),
		refreshRate: 100 * time.Millisecond,
		quiet:       false,
		barSize:     20,
	}

	// Call updateDisplay
	p.updateDisplay()

	// Close the writer to flush the buffer and get the output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify the output contains expected elements
	expectedElements := []string{
		"\r[",           // Start of progress bar
		"50.0%",         // Percentage
		"500B of 1000B", // Bytes read/total
	}

	for _, expected := range expectedElements {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', got '%s'", expected, output)
		}
	}

	// Check for transfer rate (more flexible)
	if !strings.Contains(output, "B/s") {
		t.Errorf("Expected output to contain transfer rate (B/s), got '%s'", output)
	}
}

// TestBuildProgressBarWithMockTime tests buildProgressBar with controlled time
func TestBuildProgressBarWithMockTime(t *testing.T) {
	// Create a fixed start time
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a Progress instance with the fixed start time
	p := &Progress{
		bytesRead:   0,
		totalSize:   1000,
		startTime:   startTime,
		lastUpdate:  startTime,
		refreshRate: 100 * time.Millisecond,
		quiet:       false,
		barSize:     10,
	}

	// Test at t+0s with 0 bytes read
	bar1 := p.buildProgressBar(0 * time.Second)
	if !strings.Contains(bar1, "0.0%") {
		t.Errorf("Expected 0.0%%, got %s", bar1)
	}

	// Update progress to 50% at t+5s
	p.bytesRead = 500
	bar2 := p.buildProgressBar(5 * time.Second)

	// Check percentage
	if !strings.Contains(bar2, "50.0%") {
		t.Errorf("Expected 50.0%%, got %s", bar2)
	}

	// Check transfer rate (should be around 100B/s)
	if !strings.Contains(bar2, "100B/s") {
		t.Errorf("Expected transfer rate around 100B/s, got %s", bar2)
	}

	// Update progress to 100% at t+10s
	p.bytesRead = 1000
	bar3 := p.buildProgressBar(10 * time.Second)

	// Check percentage
	if !strings.Contains(bar3, "100.0%") {
		t.Errorf("Expected 100.0%%, got %s", bar3)
	}

	// Check "Done!" message
	if !strings.Contains(bar3, "Done!") {
		t.Errorf("Expected 'Done!' message, got %s", bar3)
	}
}

// TestIndeterminateProgressWithMockTime tests indeterminate progress with controlled time
func TestIndeterminateProgressWithMockTime(t *testing.T) {
	// Create a fixed start time
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a Progress instance with indeterminate size
	p := &Progress{
		bytesRead:   1024,
		totalSize:   0, // Indeterminate
		startTime:   startTime,
		lastUpdate:  startTime,
		refreshRate: 100 * time.Millisecond,
		quiet:       false,
		barSize:     10,
	}

	// Test at different time points to verify the moving indicator
	timePoints := []time.Duration{
		0 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		400 * time.Millisecond,
		500 * time.Millisecond,
	}

	var bars []string
	for _, elapsed := range timePoints {
		bar := p.buildProgressBar(elapsed)
		bars = append(bars, bar)
	}

	// Verify that the bar changes over time (moving indicator)
	for i := 1; i < len(bars); i++ {
		if bars[i] == bars[i-1] {
			t.Errorf("Expected bar to change over time in indeterminate mode, got same bar at time points %v and %v",
				timePoints[i-1], timePoints[i])
		}
	}
}

// TestFormatSizeTable tests formatSize with a table of test cases
func TestFormatSizeTable(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0B"},
		{1, "1B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.00MB"},
		{1572864, "1.50MB"},
		{1073741824, "1.00GB"},
		{1610612736, "1.50GB"},
		{-1, "?"},
	}

	for _, test := range tests {
		result := formatSize(test.size)
		if result != test.expected {
			t.Errorf("formatSize(%d) = %s, expected %s", test.size, result, test.expected)
		}
	}
}

// TestFormatDurationTable tests formatDuration with a table of test cases
func TestFormatDurationTable(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{0, "0s"},
		{1, "1s"},
		{59, "59s"},
		{60, "1m00s"},
		{61, "1m01s"},
		{3599, "59m59s"},
		{3600, "1h00m00s"},
		{3661, "1h01m01s"},
		{86399, "23h59m59s"},
		{86400, "24h00m00s"},
	}

	for _, test := range tests {
		result := formatDuration(test.seconds)
		if result != test.expected {
			t.Errorf("formatDuration(%f) = %s, expected %s", test.seconds, result, test.expected)
		}
	}
}
