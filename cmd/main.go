package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ebitengine/oto/v3"
	wave "github.com/personal-island9353/wave-decoder/v2/internal"
)

func getOtoFormat(bitsPerSample uint16) *oto.Format {
	switch bitsPerSample {
	case 8:
		return new(oto.FormatUnsignedInt8)
	case 16:
		return new(oto.FormatSignedInt16LE)
	case 24:
		return new(oto.FormatSignedInt16LE)
	default:
		return nil
	}
}

func addPrefixSpace(s string) string {
	// Split into lines, prefix each with "\t", then join back
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file>\n", os.Args[0])
		os.Exit(1)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	parser := wave.NewParser(data)
	if err = parser.Parse(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	metadata := parser.GetMetadata()
	fmt.Println("Metadata:")
	fmt.Printf("%s\n", addPrefixSpace(metadata.String()))
	audioData := parser.GetAudioData()

	otoFormat := getOtoFormat(metadata.BitsPerSample)

	if otoFormat == nil {
		fmt.Fprintf(os.Stderr, "Unsupported bits per sample: %d\n", metadata.BitsPerSample)
		os.Exit(1)
	}

	op := &oto.NewContextOptions{
		SampleRate:   int(metadata.SampleRate),
		ChannelCount: int(metadata.NumChannels),
		Format:       *otoFormat,
	}

	otoCtx, readyChan, err := oto.NewContext(op)
	if err != nil {
		fmt.Printf("Failed to start audio context: %v\n", err)
		os.Exit(1)
	}
	<-readyChan

	fmt.Println("Playing...")
	player := otoCtx.NewPlayer(bytes.NewReader(audioData))
	player.Play()

	for player.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
	}
}
