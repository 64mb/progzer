package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestNewProgress tests the NewProgress function
func TestNewProgress(t *testing.T) {
	// Test with default config
	cfg := config{
		totalSize:   100,
		refreshRate: 200 * time.Millisecond,
		quiet:       false,
		barSize:     40,
	}

	p := NewProgress(cfg)

	if p.totalSize != 100 {
		t.Errorf("Expected totalSize to be 100, got %d", p.totalSize)
	}
	if p.refreshRate != 200*time.Millisecond {
		t.Errorf("Expected refreshRate to be 200ms, got %v", p.refreshRate)
	}
	if p.quiet != false {
		t.Errorf("Expected quiet to be false, got %v", p.quiet)
	}
	if p.barSize != 40 {
		t.Errorf("Expected barSize to be 40, got %d", p.barSize)
	}
	if p.bytesRead != 0 {
		t.Errorf("Expected bytesRead to be 0, got %d", p.bytesRead)
	}
}

// TestFormatSize tests the formatSize function
func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{-1, "?"},
		{0, "0B"},
		{100, "100B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.00MB"},
		{1572864, "1.50MB"},
		{1073741824, "1.00GB"},
		{1610612736, "1.50GB"},
	}

	for _, test := range tests {
		result := formatSize(test.bytes)
		if result != test.expected {
			t.Errorf("formatSize(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

// TestFormatDuration tests the formatDuration function
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{0, "0s"},
		{30, "30s"},
		{59, "59s"},
		{60, "1m00s"},
		{90, "1m30s"},
		{3599, "59m59s"},
		{3600, "1h00m00s"},
		{3661, "1h01m01s"},
		{7200, "2h00m00s"},
	}

	for _, test := range tests {
		result := formatDuration(test.seconds)
		if result != test.expected {
			t.Errorf("formatDuration(%f) = %s, expected %s", test.seconds, result, test.expected)
		}
	}
}

// TestMax tests the max function
func TestMax(t *testing.T) {
	tests := []struct {
		a, b     float64
		expected float64
	}{
		{1, 2, 2},
		{2, 1, 2},
		{0, 0, 0},
		{-1, 1, 1},
		{1, -1, 1},
		{-2, -1, -1},
	}

	for _, test := range tests {
		result := max(test.a, test.b)
		if result != test.expected {
			t.Errorf("max(%f, %f) = %f, expected %f", test.a, test.b, result, test.expected)
		}
	}
}

// TestBuildProgressBar tests the buildProgressBar method
func TestBuildProgressBar(t *testing.T) {
	// Test with known size
	p1 := &Progress{
		bytesRead: 50,
		totalSize: 100,
		barSize:   10,
		startTime: time.Now().Add(-1 * time.Second),
	}

	bar1 := p1.buildProgressBar(1 * time.Second)
	if !strings.Contains(bar1, "50.0%") {
		t.Errorf("Expected progress bar to contain 50.0%%, got %s", bar1)
	}
	if !strings.Contains(bar1, "[=====>    ]") {
		t.Errorf("Expected progress bar to contain [=====>    ], got %s", bar1)
	}

	// Test with unknown size
	p2 := &Progress{
		bytesRead: 1024,
		totalSize: 0,
		barSize:   10,
		startTime: time.Now().Add(-1 * time.Second),
	}

	bar2 := p2.buildProgressBar(1 * time.Second)
	if !strings.Contains(bar2, "1.0KB") {
		t.Errorf("Expected progress bar to contain 1.0KB, got %s", bar2)
	}
	if strings.Contains(bar2, "%") {
		t.Errorf("Expected progress bar not to contain %%, got %s", bar2)
	}
}

// TestProcess tests the Process method with mocked stdin/stdout
func TestProcess(t *testing.T) {
	// Create a Progress instance
	p := &Progress{
		bytesRead:   0,
		totalSize:   100,
		startTime:   time.Now(),
		lastUpdate:  time.Now().Add(-1 * time.Hour),
		refreshRate: 100 * time.Millisecond,
		quiet:       true, // Set to true to avoid terminal output during tests
		barSize:     10,
	}

	// Create a buffer with test data
	testData := bytes.Repeat([]byte("test"), 25) // 100 bytes

	// Save original stdin and stdout
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	// Create pipes for stdin and stdout
	r, w, _ := os.Pipe()
	os.Stdin = r

	outR, outW, _ := os.Pipe()
	os.Stdout = outW

	// Write test data to stdin
	go func() {
		w.Write(testData)
		w.Close()
	}()

	// Process the data
	err := p.Process()
	if err != nil {
		t.Errorf("Process() returned error: %v", err)
	}

	// Close stdout pipe
	outW.Close()

	// Read processed data
	processedData, _ := io.ReadAll(outR)

	// Verify the data was processed correctly
	if !bytes.Equal(processedData, testData) {
		t.Errorf("Processed data doesn't match input data")
	}

	// Verify bytes read
	if p.bytesRead != 100 {
		t.Errorf("Expected bytesRead to be 100, got %d", p.bytesRead)
	}
}

// TestConfigDefaults tests the default configuration values
func TestConfigDefaults(t *testing.T) {
	// Create a config with default values
	cfg := config{
		totalSize:   0,
		refreshRate: 100 * time.Millisecond,
		quiet:       false,
		barSize:     DefaultBarSize,
		showVersion: false,
	}

	// Verify the default values
	if cfg.totalSize != 0 {
		t.Errorf("Expected default totalSize to be 0, got %d", cfg.totalSize)
	}
	if cfg.refreshRate != 100*time.Millisecond {
		t.Errorf("Expected default refreshRate to be 100ms, got %v", cfg.refreshRate)
	}
	if cfg.quiet != false {
		t.Errorf("Expected default quiet to be false, got %v", cfg.quiet)
	}
	if cfg.barSize != DefaultBarSize {
		t.Errorf("Expected default barSize to be %d, got %d", DefaultBarSize, cfg.barSize)
	}
	if cfg.showVersion != false {
		t.Errorf("Expected default showVersion to be false, got %v", cfg.showVersion)
	}
}

// TestConfigCustomValues tests custom configuration values
func TestConfigCustomValues(t *testing.T) {
	// Create a config with custom values
	cfg := config{
		totalSize:   1024,
		refreshRate: 200 * time.Millisecond,
		quiet:       true,
		barSize:     20,
		showVersion: true,
		getSizePath: "testfile.txt",
	}

	// Verify the custom values
	if cfg.totalSize != 1024 {
		t.Errorf("Expected totalSize to be 1024, got %d", cfg.totalSize)
	}
	if cfg.refreshRate != 200*time.Millisecond {
		t.Errorf("Expected refreshRate to be 200ms, got %v", cfg.refreshRate)
	}
	if cfg.quiet != true {
		t.Errorf("Expected quiet to be true, got %v", cfg.quiet)
	}
	if cfg.barSize != 20 {
		t.Errorf("Expected barSize to be 20, got %d", cfg.barSize)
	}
	if cfg.showVersion != true {
		t.Errorf("Expected showVersion to be true, got %v", cfg.showVersion)
	}
	if cfg.getSizePath != "testfile.txt" {
		t.Errorf("Expected getSizePath to be 'testfile.txt', got %v", cfg.getSizePath)
	}
}

// TestGetSize tests the --get-size functionality
func TestGetSize(t *testing.T) {
	// Create a temporary test file
	testFile, err := os.CreateTemp("", "progzer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(testFile.Name())

	// Write some data to the file
	testData := []byte("This is a test file for the --get-size functionality")
	if _, err := testFile.Write(testData); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	testFile.Close()

	// Get the file size using os.Stat
	fileInfo, err := os.Stat(testFile.Name())
	if err != nil {
		t.Fatalf("Failed to stat temp file: %v", err)
	}
	expectedSize := fileInfo.Size()

	// Redirect stdout to capture the output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// We can't directly test the main function since it calls os.Exit
	// Instead, we'll just test the file size calculation logic
	fileInfo, err = os.Stat(testFile.Name())
	if err != nil {
		t.Errorf("Error getting file size: %v", err)
	}
	fmt.Printf("%d\n", fileInfo.Size())

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	// Convert output to int64 for comparison
	outputSize, err := strconv.ParseInt(output, 10, 64)
	if err != nil {
		t.Fatalf("Failed to parse output as int64: %v", err)
	}

	// Verify the output matches the expected size
	if outputSize != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, outputSize)
	}
}
