package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/cmplx"
	"os"
	"strconv"

	"github.com/hajimehoshi/go-mp3"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
)

func process(toneFreq int, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	d, err := mp3.NewDecoder(f)
	if err != nil {
		return err
	}
	defer d.Close()

	windowSize := 40 // window size in milliseconds
	windowSamples := int(float32(d.SampleRate()) * float32(windowSize) / 1000.0)

	// fftSize is the smallest power of 2 greater than or equal to windowSamples
	fftSize := int(math.Pow(2, math.Ceil(math.Log2(float64(windowSamples)))))

	spectralWidth := float64(d.SampleRate()) / float64(fftSize)
	targetIndex := int(float64(toneFreq) / spectralWidth)

	fmt.Printf("Sample Rate: %d\n", d.SampleRate())
	fmt.Printf("Length: %d[bytes]\n", d.Length())
	fmt.Printf("Window size: %d[samples]\n", windowSamples)
	fmt.Printf("FFT size: %d\n", fftSize)
	fmt.Printf("Spectral Line width: %v[hertz]\n", spectralWidth)
	fmt.Printf("Tone index: %d\n", targetIndex)

	b := make([]byte, windowSamples*4) // 2 bytes per sample, 2 channels
	w := make([]float64, fftSize)
	t := 0
	toneStart := -1

outerloop:
	for {
		// Read a window of samples
		bytesRead := 0
		for bytesRead < len(b) {
			n, err := d.Read(b[bytesRead:])
			if err != nil {
				break outerloop
			}
			bytesRead += n
		}

		// Convert to float (ignore second channel)
		for i := 0; i < len(b); i += 4 {
			w[i/4] = float64(int16(binary.LittleEndian.Uint16(b[i+0:i+2]))) / 32768.0
		}

		// Apply window function
		window.Apply(w, window.Hamming)

		// Perform FFT
		c := fft.FFTReal(w)

		// Compute the normalized magnitude
		r, _ := cmplx.Polar(c[targetIndex])
		r = r / float64(fftSize)

		// Look for tone
		toneDetected := r > 0.05 // Apply arbitrary threshold
		if toneDetected && toneStart < 0 {
			toneStart = t
		} else if !toneDetected && (toneStart >= 0) {
			fmt.Printf("Tone from %dms to %dms.\n", toneStart, t)
			toneStart = -1
		}

		t += windowSize
	}

	return nil
}

func main() {
	toneFreq, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("Usage: %s <tone frequency> <mp3 filename>\n", os.Args[0])
		return
	}
	if err := process(toneFreq, os.Args[2]); err != nil {
		log.Fatal(err)
	}
}
