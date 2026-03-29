package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
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
	fmt.Printf("%s\n", metadata)
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

	player := otoCtx.NewPlayer(bytes.NewReader(audioData))
	player.Play()

	for player.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
	}
}
