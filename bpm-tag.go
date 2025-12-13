package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/go-mp3"
)

const (
	RATE     = 44100.0
	LOWER    = 84.0
	UPPER    = 180.0
	BLOCK    = 4096
	INTERVAL = 128
)

// sample from the metered energy
func sample(nrg []float64, offset float64) float64 {
	n := math.Floor(offset)
	i := int(n)

	if n >= 0.0 && i < len(nrg) {
		return nrg[i]
	}
	return 0.0
}

// autodifference tests an interval for the given energy data
func autodifference(nrg []float64, interval float64) float64 {
	beats := []float64{-32, -16, -8, -4, -2, -1, 1, 2, 4, 8, 16, 32}
	nobeats := []float64{-0.5, -0.25, 0.25, 0.5}

	mid := rand.Float64() * float64(len(nrg))
	v := sample(nrg, mid)

	diff := 0.0
	total := 0.0

	for _, beat := range beats {
		y := sample(nrg, mid+beat*interval)
		w := 1.0 / math.Abs(beat)
		diff += w * math.Abs(y-v)
		total += w
	}

	for _, nobeat := range nobeats {
		y := sample(nrg, mid+nobeat*interval)
		w := math.Abs(nobeat)
		diff -= w * math.Abs(y-v)
		total += w
	}

	return diff / total
}

// bpmToInterval converts beats-per-minute to a sampling interval in energy space
func bpmToInterval(bpm float64) float64 {
	beatsPerSecond := bpm / 60.0
	samplesPerBeat := RATE / beatsPerSecond
	return samplesPerBeat / INTERVAL
}

// intervalToBPM converts sampling interval in energy space to beats-per-minute
func intervalToBPM(interval float64) float64 {
	samplesPerBeat := interval * INTERVAL
	beatsPerSecond := RATE / samplesPerBeat
	return beatsPerSecond * 60.0
}

// scanForBPM scans a range of BPM values for the one with minimum autodifference
func scanForBPM(nrg []float64, slowest, fastest float64, steps, samples int) float64 {
	slowest = bpmToInterval(slowest)
	fastest = bpmToInterval(fastest)
	step := (slowest - fastest) / float64(steps)

	height := math.Inf(1)
	trough := math.NaN()

	for interval := fastest; interval <= slowest; interval += step {
		t := 0.0
		for s := 0; s < samples; s++ {
			t += autodifference(nrg, interval)
		}

		if t < height {
			trough = interval
			height = t
		}
	}

	return intervalToBPM(trough)
}

// convertToMono converts stereo samples to mono
func convertToMono(data []byte, channels int) []float32 {
	if channels == 1 {
		samples := make([]float32, len(data)/2)
		for i := 0; i < len(samples); i++ {
			// Convert 16-bit PCM to float32
			sample := int16(data[i*2]) | int16(data[i*2+1])<<8
			samples[i] = float32(sample) / 32768.0
		}
		return samples
	}

	// Stereo to mono: average left and right channels
	numSamples := len(data) / 4
	samples := make([]float32, numSamples)
	for i := 0; i < numSamples; i++ {
		left := int16(data[i*4]) | int16(data[i*4+1])<<8
		right := int16(data[i*4+2]) | int16(data[i*4+3])<<8
		samples[i] = (float32(left) + float32(right)) / 2.0 / 32768.0
	}
	return samples
}

func analyzeBPM(filename string, minBPM, maxBPM float64) (float64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return 0, err
	}

	sampleRate := float64(decoder.SampleRate())

	// Calculate the interval adjusted for the actual sample rate
	// We want to sample energy at the same time intervals regardless of sample rate
	actualInterval := int(math.Round(INTERVAL * sampleRate / RATE))
	if actualInterval < 1 {
		actualInterval = 1
	}

	var nrg []float64
	v := 0.0
	n := 0

	buf := make([]byte, 4096)
	for {
		bytesRead, err := decoder.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}

		samples := convertToMono(buf[:bytesRead], 2)

		for _, z := range samples {
			// Maintain an energy meter (similar to PPM)
			absZ := math.Abs(float64(z))
			if absZ > v {
				v += (absZ - v) / 8.0
			} else {
				v -= (v - absZ) / 512.0
			}

			// At regular intervals, sample the energy
			n++
			if n >= actualInterval {
				n = 0
				nrg = append(nrg, v)
			}
		}
	}

	if len(nrg) == 0 {
		return 0, fmt.Errorf("no energy data collected")
	}

	bpm := scanForBPM(nrg, minBPM, maxBPM, 1024, 1024)

	return bpm, nil
}

func main() {
	minBPM := flag.Float64("m", LOWER, "Minimum detected BPM")
	maxBPM := flag.Float64("x", UPPER, "Maximum detected BPM")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: bpm-tag [options] <file.mp3>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	filename := flag.Arg(0)

	rand.Seed(time.Now().UnixNano())

	if *verbose {
		file, err := os.Open(filename)
		if err == nil {
			decoder, err := mp3.NewDecoder(file)
			if err == nil {
				fmt.Fprintf(os.Stderr, "Sample rate: %d Hz\n", decoder.SampleRate())
			}
			file.Close()
		}
	}

	bpm, err := analyzeBPM(filename, *minBPM, *maxBPM)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%.3f\n", bpm)
}
