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

	for i := uint32(12); i < chunkSize; i++ {
		// TODO: parse file content
	}
}
