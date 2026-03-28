package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		_, err := fmt.Fprintf(os.Stderr, "Usage: %s <file>\n", os.Args[0])
		if err != nil {
			os.Exit(1)
			return
		}
		os.Exit(1)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Println(err)
	}

	if len(data) < 12 {
		_, err := fmt.Fprintf(os.Stderr, "File %s is too small\n", os.Args[0])
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	riff := string(data[:4])
	if riff != "RIFF" {
		_, err := fmt.Fprintf(os.Stderr, "File %s contains invalid RIFF header\n", os.Args[0])
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	chunkSize := binary.LittleEndian.Uint32(data[4:8])

	format := string(data[8:12])
	if format != "WAVE" {
		_, err := fmt.Fprintf(os.Stderr, "File %s contains invalid WAVE format\n", os.Args[0])
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	chunkId := string(data[12 : 12+4])
	if chunkId != "fmt " {
		_, err := fmt.Fprintf(os.Stderr, "File %s misses fmt chunk\n", os.Args[0])
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	fmtChunkSize := binary.LittleEndian.Uint32(data[16:20])

	if uint32(len(data)) < 20+fmtChunkSize {
		_, err := fmt.Fprintf(os.Stderr, "File %s contains malformed fmt chunk\n", os.Args[0])
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	audioFormat := binary.LittleEndian.Uint16(data[20:22])
	if audioFormat != 1 {
		_, err := fmt.Fprintf(os.Stderr, "File %s contains unsupported audio format\n", os.Args[0])
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	numChannels := binary.LittleEndian.Uint16(data[22:24])
	if numChannels != 1 {
		_, err := fmt.Fprintf(os.Stderr, "File %s contains unsupported number of channels %s\n", os.Args[0], numChannels)
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	fmt.Printf("Sample rate: %d\n", sampleRate)

	byteRate := binary.LittleEndian.Uint32(data[28:32])
	fmt.Printf("Byte rate: %d\n", byteRate)

	blockAlign := binary.LittleEndian.Uint16(data[32:34])
	fmt.Printf("Block align: %d\n", blockAlign)

	bitsPerSample := binary.LittleEndian.Uint16(data[34:36])
	fmt.Printf("Bits per sample: %d\n", bitsPerSample)

	if byteRate != sampleRate*uint32(numChannels)*uint32(bitsPerSample)/8 {
		_, err := fmt.Fprintf(os.Stderr, "File %s contains malformed byte rate\n", os.Args[0])
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	if blockAlign != numChannels*bitsPerSample/8 {
		_, err := fmt.Fprintf(os.Stderr, "File %s contains malformed block align\n", os.Args[0])
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	for i := uint32(36); i < chunkSize; i++ {
		chunkId := string(data[i : i+4])

		if chunkId == "data" {
			// TODO: decode samples based on format
		}
	}
}
