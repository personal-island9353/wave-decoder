package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ebitengine/oto/v3"
)

type decoderFunc func(data []byte) []byte

func decode8Bit(data []byte) []byte {
	return data
}

func decode16Bit(data []byte) []byte {
	return data
}

func decode24Bit(data []byte) []byte {
	var decodedData []byte
	for i := 0; i+3 <= len(data); i += 3 {
		b1 := uint32(data[i])
		b2 := uint32(data[i+1])
		b3 := uint32(data[i+2])

		val24 := b1 | (b2 << 8) | (b3 << 16)
		if val24&0x800000 != 0 {
			val24 |= 0xFF000000
		}
		val16 := int16(int32(val24) >> 8)
		decodedData = append(decodedData, byte(val16), byte(val16>>8))
	}
	return decodedData
}

func getDecoder(bitsPerSample uint16) (decoderFunc, *oto.Format) {
	switch bitsPerSample {
	case 8:
		return decode8Bit, new(oto.FormatUnsignedInt8)
	case 16:
		return decode16Bit, new(oto.FormatSignedInt16LE)
	case 24:
		return decode24Bit, new(oto.FormatSignedInt16LE)
	default:
		return nil, nil
	}
}

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

	_ = binary.LittleEndian.Uint32(data[4:8]) // Read RIFF chunk size but don't strictly use it as we scan the whole slice

	waveFormat := string(data[8:12])
	if waveFormat != "WAVE" {
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
	if numChannels != 1 && numChannels != 2 {
		_, err := fmt.Fprintf(os.Stderr, "File %s contains unsupported number of channels %d\n", os.Args[0], numChannels)
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

	var audioData []byte
	var otoFormat oto.Format
	chunkOffset := 20 + fmtChunkSize

	decoder, formatPtr := getDecoder(bitsPerSample)
	if formatPtr == nil {
		_, err := fmt.Fprintf(os.Stderr, "Unsupported bits per sample: %d\n", bitsPerSample)
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}
	otoFormat = *formatPtr

	for chunkOffset+8 <= uint32(len(data)) {
		chunkId := string(data[chunkOffset : chunkOffset+4])
		chunkSize := binary.LittleEndian.Uint32(data[chunkOffset+4 : chunkOffset+8])

		if chunkId == "data" {
			dataStart := chunkOffset + 8
			dataEnd := dataStart + chunkSize
			if dataEnd > uint32(len(data)) {
				dataEnd = uint32(len(data))
			}

			audioData = decoder(data[dataStart:dataEnd])
			break
		}
		chunkOffset += 8 + chunkSize
	}

	if len(audioData) == 0 {
		_, err := fmt.Fprintln(os.Stderr, "Data chunks not found")
		if err != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}

	op := &oto.NewContextOptions{
		SampleRate:   int(sampleRate),
		ChannelCount: int(numChannels),
		Format:       otoFormat,
	}

	otoCtx, readyChan, err := oto.NewContext(op)
	if err != nil {
		fmt.Printf("oto.NewContext failed: %v\n", err)
		os.Exit(1)
	}
	<-readyChan

	player := otoCtx.NewPlayer(bytes.NewReader(audioData))
	player.Play()

	for player.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
	}
}
