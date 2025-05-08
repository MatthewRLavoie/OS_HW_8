package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Log Entry Structure
////////////////////////////////////////////////////////////////////////////////

// LogEntry represents a single log line with timestamp, level, context, and message
type LogEntry struct {
	Timestamp string
	Level     string
	Context   string
	Message   string
}

// String converts a LogEntry to the required string format
func (e LogEntry) String() string {
	return fmt.Sprintf("[%s] [%s] [%s] %s", e.Timestamp, e.Level, e.Context, e.Message)
}

////////////////////////////////////////////////////////////////////////////////
// Naive Logger: No synchronization, uses fsync after every write
////////////////////////////////////////////////////////////////////////////////

type NaiveLogger struct {
	file *os.File
}

// Creates a new naive logger
func NewNaiveLogger(filename string) *NaiveLogger {
	f, _ := os.Create(filename)
	return &NaiveLogger{file: f}
}

// Writes directly to file with no synchronization
func (l *NaiveLogger) Log(entry LogEntry) {
	l.file.WriteString(entry.String() + "\n")
	l.file.Sync() // fsync after every write
}

// Closes the file
func (l *NaiveLogger) Close() {
	l.file.Close()
}

////////////////////////////////////////////////////////////////////////////////
// Mutex Logger: Uses sync.Mutex and batches fsync every 10 entries
////////////////////////////////////////////////////////////////////////////////

type MutexLogger struct {
	file   *os.File
	mu     sync.Mutex // global mutex for thread-safe writes
	buffer []string   // holds log lines before batch write
	count  int        // count of buffered log entries
}

// Creates a mutex logger with internal buffer
func NewMutexLogger(filename string) *MutexLogger {
	f, _ := os.Create(filename)
	return &MutexLogger{file: f, buffer: make([]string, 0, 10)}
}

// Logs a new entry, synchronized with mutex, batching every 10 entries
func (l *MutexLogger) Log(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buffer = append(l.buffer, entry.String()+"\n")
	l.count++

	if l.count >= 10 {
		for _, line := range l.buffer {
			l.file.WriteString(line)
		}
		l.file.Sync() // fsync after batch
		l.buffer = l.buffer[:0]
		l.count = 0
	}
}

// Final flush of buffer and close
func (l *MutexLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, line := range l.buffer {
		l.file.WriteString(line)
	}
	l.file.Sync()
	l.file.Close()
}

////////////////////////////////////////////////////////////////////////////////
// Channel Logger: Uses a goroutine and channel, batches fsync every 10 entries
////////////////////////////////////////////////////////////////////////////////

type ChannelLogger struct {
	ch chan LogEntry  // log entries go into this channel
	wg sync.WaitGroup // for waiting on goroutine shutdown
}

// Creates a channel-based logger with a background goroutine for file writes
func NewChannelLogger(filename string) *ChannelLogger {
	ch := make(chan LogEntry, 100)
	logger := &ChannelLogger{ch: ch}
	logger.wg.Add(1)

	go func() {
		defer logger.wg.Done()
		file, _ := os.Create(filename)
		defer file.Close()

		buffer := make([]string, 0, 10)
		count := 0

		for entry := range ch {
			buffer = append(buffer, entry.String()+"\n")
			count++

			if count >= 10 {
				for _, line := range buffer {
					file.WriteString(line)
				}
				file.Sync()
				buffer = buffer[:0]
				count = 0
			}
		}

		// Flush any remaining entries on shutdown
		for _, line := range buffer {
			file.WriteString(line)
		}
		file.Sync()
	}()

	return logger
}

// Sends a log entry to the background goroutine
func (l *ChannelLogger) Log(entry LogEntry) {
	l.ch <- entry
}

// Closes the channel and waits for logger to finish
func (l *ChannelLogger) Close() {
	close(l.ch)
	l.wg.Wait()
}

////////////////////////////////////////////////////////////////////////////////
// Benchmark Function: Simulates concurrent logging from 10 goroutines
////////////////////////////////////////////////////////////////////////////////

// Runs the logger with 10 goroutines writing 50 log entries each
func simulateLogging(logger interface {
	Log(LogEntry)
}, done func()) time.Duration {
	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				entry := LogEntry{
					Timestamp: time.Now().Format("2006-01-02 15:04:05"),
					Level:     "INFO",
					Context:   fmt.Sprintf("req-%d", id),
					Message:   fmt.Sprintf("message %d", j),
				}
				logger.Log(entry)
			}
		}(i)
	}

	wg.Wait()
	if done != nil {
		done()
	}
	return time.Since(start)
}

////////////////////////////////////////////////////////////////////////////////
// Main Function: Run and benchmark all three logger implementations
////////////////////////////////////////////////////////////////////////////////

func main() {
	// Run and time the naive logger
	naive := NewNaiveLogger("naive.log")
	fmt.Println("Naive Logger:", simulateLogging(naive, naive.Close))

	// Run and time the mutex logger
	mutex := NewMutexLogger("mutex.log")
	fmt.Println("Mutex Logger:", simulateLogging(mutex, mutex.Close))

	// Run and time the channel logger
	channel := NewChannelLogger("channel.log")
	fmt.Println("Channel Logger:", simulateLogging(channel, channel.Close))
}
