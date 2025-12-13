# bpm-tag

  Two Versions Available:

  bpm-tag.go - A standalone Go program that:
  - Takes an MP3 file as a command-line argument
  - Decodes the MP3 directly (no need for external tools like sox)
  - Implements the same BPM detection algorithm as the original C program
  - Outputs only the BPM value in the format: ###.###
  - Uses go-mp3 library for MP3 decoding
  - No external dependencies besides the Go library
  - Handles variable sample rates internally

  bpm-tag-v2 (Hybrid - bpm-tag-v2.go:1-177)
  - Uses sox for MP3 decoding (like the original script)
  - More reliable for edge cases
  - Requires sox to be installed
  - Guaranteed to match the C implementation's audio processing

  Both versions implement the same BPM detection algorithm in pure Go.

  Key features:
  - Uses the go-mp3 library for MP3 decoding
  - Implements the energy meter and autodifference algorithm
  - Supports -m (minimum BPM) and -x (maximum BPM) flags
  - Default BPM range: 84-146

  Usage:

  ```
  ./bpm-tag-go song.mp3
  # Output: 128.456
  ```

  # With custom BPM range:

  ```
  ./bpm-tag-go -m 120 -x 180 song.mp3
  ```

  The compiled binary bpm-tag-go is ready to use. The program faithfully replicates the BPM detection logic from the original C code while
  handling MP3 files directly.

# Credits

The [Original Program](http://www.pogo.org.uk/~mark/bpm-tools/) was invented by Mark Hills <mark(at)xwax.org>
