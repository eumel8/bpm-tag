package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"time"
)

const (
	RATE     = 44100.0
	LOWER    = 84.0
	UPPER    = 146.0
	INTERVAL = 128
)

func sample(nrg []float64, offset float64) float64 {
	n := math.Floor(offset)
	i := int(n)

	if n >= 0.0 && i < len(nrg) {
		return nrg[i]
	}
	return 0.0
}

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

func bpmToInterval(bpm float64) float64 {
	beatsPerSecond := bpm / 60.0
	samplesPerBeat := RATE / beatsPerSecond
	return samplesPerBeat / INTERVAL
}

func intervalToBPM(interval float64) float64 {
	samplesPerBeat := interval * INTERVAL
	beatsPerSecond := RATE / samplesPerBeat
	return beatsPerSecond * 60.0
}

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

// testBPM calculates the autodifference score for a specific BPM
func testBPM(nrg []float64, bpm float64, samples int) float64 {
	interval := bpmToInterval(bpm)
	t := 0.0
	for s := 0; s < samples; s++ {
		t += autodifference(nrg, interval)
	}
	return t / float64(samples)
}

// checkHarmonics verifies if the detected BPM should be halved or doubled
func checkHarmonics(nrg []float64, bpm float64) float64 {
	score := testBPM(nrg, bpm, 64)
	scoreHalf := testBPM(nrg, bpm/2.0, 64)
	scoreDouble := testBPM(nrg, bpm*2.0, 64)

	// Prefer BPM in the 85-170 range (most common for music)
	// If half the BPM is close in score and in preferred range, use it
	if bpm > 160 && bpm/2.0 >= 80 {
		// If halved BPM has similar or better score, prefer it
		if scoreHalf <= score*1.1 {
			return bpm / 2.0
		}
	}

	// If double the BPM is close in score and in preferred range, use it
	if bpm < 90 && bpm*2.0 <= 180 {
		// If doubled BPM has similar or better score, prefer it
		if scoreDouble <= score*1.1 {
			return bpm * 2.0
		}
	}

	return bpm
}

func analyzeBPM(reader io.Reader, minBPM, maxBPM float64) (float64, error) {
	var nrg []float64
	v := 0.0
	n := 0

	for {
		var z float32
		err := binary.Read(reader, binary.LittleEndian, &z)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}

		// Maintain an energy meter (similar to PPM)
		absZ := math.Abs(float64(z))
		if absZ > v {
			v += (absZ - v) / 8.0
		} else {
			v -= (v - absZ) / 512.0
		}

		// At regular intervals, sample the energy
		n++
		if n == INTERVAL {
			n = 0
			nrg = append(nrg, v)
		}
	}

	if len(nrg) == 0 {
		return 0, fmt.Errorf("no energy data collected")
	}

	bpm := scanForBPM(nrg, minBPM, maxBPM, 1024, 1024)
	bpm = checkHarmonics(nrg, bpm)
	return bpm, nil
}

func main() {
	minBPM := flag.Float64("m", LOWER, "Minimum detected BPM")
	maxBPM := flag.Float64("x", UPPER, "Maximum detected BPM")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: bpm-tag [options] <file.mp3>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	filename := flag.Arg(0)

	rand.Seed(time.Now().UnixNano())

	// Use sox to decode MP3 to raw 32-bit float PCM at 44100 Hz, mono
	cmd := exec.Command("sox", "-V1", filename, "-r", "44100", "-e", "float", "-c", "1", "-t", "raw", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating pipe: %v\n", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting sox: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure 'sox' is installed\n")
		os.Exit(1)
	}

	bpm, err := analyzeBPM(stdout, *minBPM, *maxBPM)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		cmd.Wait()
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Error from sox: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%.3f\n", bpm)
}
