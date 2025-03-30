package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Version information
const (
	Version        = "1.0.0"
	DefaultBarSize = 34 // Default progress bar size in characters
)

// Configuration options
type config struct {
	totalSize   int64
	refreshRate time.Duration
	quiet       bool
	barSize     int
	showVersion bool
}

// Progress holds the state of the progress bar
type Progress struct {
	bytesRead   int64
	totalSize   int64
	startTime   time.Time
	lastUpdate  time.Time
	refreshRate time.Duration
	quiet       bool
	barSize     int
}

func main() {
	// Parse command line flags
	cfg := parseFlags()

	// Show version and exit if requested
	if cfg.showVersion {
		fmt.Printf("version: %s\n", Version)
		os.Exit(0)
	}

	// Create a new progress bar
	progress := NewProgress(cfg)

	// Set up signal handling for graceful cleanup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Fprintln(os.Stderr, "\nInterrupted")
		os.Exit(1)
	}()

	// Process the data
	if err := progress.Process(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// Parse command line flags
func parseFlags() config {
	var cfg config
	flag.Int64Var(&cfg.totalSize, "size", 0, "Expected total size in bytes (default: indeterminate)")
	flag.DurationVar(&cfg.refreshRate, "refresh", 100*time.Millisecond, "Refresh rate for progress updates")
	flag.BoolVar(&cfg.quiet, "quiet", false, "Don't show progress bar")
	flag.IntVar(&cfg.barSize, "bar-size", DefaultBarSize, "Size of the progress bar in characters")
	flag.BoolVar(&cfg.showVersion, "version", false, "Show version information and exit")
	flag.Parse()

	return cfg
}

// NewProgress creates a new progress bar
func NewProgress(cfg config) *Progress {
	return &Progress{
		bytesRead:   0,
		totalSize:   cfg.totalSize,
		startTime:   time.Now(),
		lastUpdate:  time.Now().Add(-1 * time.Hour), // Force initial update
		refreshRate: cfg.refreshRate,
		quiet:       cfg.quiet,
		barSize:     cfg.barSize,
	}
}

// Process reads from stdin and writes to stdout while tracking progress
func (p *Progress) Process() error {
	// Use a larger buffer for better performance
	reader := bufio.NewReaderSize(os.Stdin, 64*1024)
	writer := bufio.NewWriterSize(os.Stdout, 64*1024)
	defer writer.Flush()

	buffer := make([]byte, 64*1024)
	ticker := time.NewTicker(p.refreshRate)
	defer ticker.Stop()

	// Create a done channel for the ticker routine
	done := make(chan struct{})
	defer close(done)

	// Update the progress bar in the background
	if !p.quiet {
		go func() {
			for {
				select {
				case <-ticker.C:
					p.updateDisplay()
				case <-done:
					return
				}
			}
		}()
	}

	// Main read/write loop
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			p.bytesRead += int64(n)
			if _, writeErr := writer.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf("error writing to stdout: %w", writeErr)
			}

			// Flush periodically to ensure data flows through the pipe
			if p.bytesRead%int64(1024*1024) == 0 {
				if flushErr := writer.Flush(); flushErr != nil {
					return fmt.Errorf("error flushing output: %w", flushErr)
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading from stdin: %w", err)
		}
	}

	// Ensure final update shows 100%
	if !p.quiet {
		p.updateDisplay()
		fmt.Fprintln(os.Stderr, "") // Final newline
	}

	return nil
}

// updateDisplay updates the progress display
func (p *Progress) updateDisplay() {
	now := time.Now()
	elapsed := now.Sub(p.startTime)

	// Build progress bar
	bar := p.buildProgressBar(elapsed)

	// Print the bar
	fmt.Fprint(os.Stderr, "\r"+bar)
}

// buildProgressBar creates the progress bar string
func (p *Progress) buildProgressBar(elapsed time.Duration) string {
	// Calculate percentages and rates
	percentComplete := 0.0
	if p.totalSize > 0 {
		percentComplete = float64(p.bytesRead) / float64(p.totalSize) * 100.0
		if percentComplete > 100.0 {
			percentComplete = 100.0
		}
	}

	// Calculate transfer rate
	bytesPerSec := float64(p.bytesRead) / max(elapsed.Seconds(), 0.001)

	// Format strings
	var completionStr string
	if p.totalSize > 0 {
		completionStr = fmt.Sprintf("%.1f%%", percentComplete)
	} else {
		completionStr = "---"
	}

	readStr := formatSize(p.bytesRead)
	totalStr := formatSize(p.totalSize)
	rateStr := fmt.Sprintf("%s/s", formatSize(int64(bytesPerSec)))

	// Calculate estimated time remaining
	var etaStr string
	if p.totalSize > 0 && p.bytesRead > 0 && bytesPerSec > 0 {
		bytesRemaining := p.totalSize - p.bytesRead
		if bytesRemaining > 0 {
			secondsRemaining := float64(bytesRemaining) / bytesPerSec
			etaStr = fmt.Sprintf(" ETA: %s", formatDuration(secondsRemaining))
		} else {
			etaStr = " Done!"
		}
	}

	// Build status text
	var statusText string
	if p.totalSize > 0 {
		statusText = fmt.Sprintf("%s of %s (%s) @ %s%s", readStr, totalStr, completionStr, rateStr, etaStr)
	} else {
		statusText = fmt.Sprintf("%s @ %s", readStr, rateStr)
	}

	// Use fixed bar size
	barWidth := p.barSize

	// Create the progress bar
	var bar strings.Builder
	bar.WriteString("[")

	if p.totalSize > 0 {
		// Known size mode
		completedWidth := int(float64(barWidth) * percentComplete / 100.0)
		if completedWidth > barWidth {
			completedWidth = barWidth
		}

		for i := 0; i < completedWidth; i++ {
			bar.WriteString("=")
		}

		if completedWidth < barWidth {
			bar.WriteString(">")
			for i := completedWidth + 1; i < barWidth; i++ {
				bar.WriteString(" ")
			}
		}
	} else {
		// Indeterminate mode - show a moving block
		position := int(elapsed.Milliseconds()/100) % (barWidth * 2)
		if position >= barWidth {
			position = barWidth*2 - position
		}

		for i := 0; i < barWidth; i++ {
			if i == position {
				bar.WriteString("=>")
				i++
			} else {
				bar.WriteString(" ")
			}
		}
	}

	bar.WriteString("] ")
	bar.WriteString(statusText)

	return bar.String()
}

// formatSize formats bytes to human-readable string
func formatSize(bytes int64) string {
	if bytes < 0 {
		return "?"
	}
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.2fMB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.2fGB", float64(bytes)/(1024*1024*1024))
}

// formatDuration formats seconds into a human-readable duration string
func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%dm%02ds", int(seconds)/60, int(seconds)%60)
	} else {
		return fmt.Sprintf("%dh%02dm%02ds", int(seconds)/3600, (int(seconds)%3600)/60, int(seconds)%60)
	}
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
