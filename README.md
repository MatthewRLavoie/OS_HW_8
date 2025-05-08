# OS_HW_8

1. Analysis
In this assignment, three different logger implementations were benchmarked to observe the effects of synchronization and fsync() on performance and correctness. The Naive Logger, which performs unsynchronized writes and calls fsync() after every entry, exhibited the worst performance at 1.36 seconds and showed evidence of interleaved or corrupted output due to concurrent access. Additionally, running with go run -race confirms the presence of data races. In contrast, the Mutex Logger achieved a significantly faster runtime of 132.5 milliseconds by protecting writes with a global mutex and batching fsync() calls after every 10 entries. This eliminated race conditions but introduced some lock contention. The Channel Logger took 135.9 milliseconds, matching the Mutex Logger in performance while offering better scalability. It uses a buffered channel and a dedicated goroutine to serialize all file access, avoiding locks altogether. These results highlight the importance of proper synchronization for correctness and how batching fsync() calls drastically improves performance. While fsync() ensures data durability, it is expensive because it forces the OS to flush file buffers to disk, making it crucial to minimize its frequency in high-throughput systems.


AI Usage - ChatGPT was used for help with making the benchmark test
