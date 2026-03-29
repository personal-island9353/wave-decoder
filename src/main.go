package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
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
		os.Exit(1)
	}

	reader := bytes.NewReader(data)

	var riff [4]byte
	if _, err := io.ReadFull(reader, riff[:]); err != nil || string(riff[:]) != "RIFF" {
		fmt.Fprintf(os.Stderr, "File %s is too small or contains invalid RIFF header\n", args[0])
		os.Exit(1)
	}

	var riffChunkSize uint32
	if err = binary.Read(reader, binary.LittleEndian, &riffChunkSize); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading Riff chunk size from %s\n", args[0])
		os.Exit(1)
	}

	var waveFormat [4]byte
	if _, err := io.ReadFull(reader, waveFormat[:]); err != nil || string(waveFormat[:]) != "WAVE" {
		fmt.Fprintf(os.Stderr, "File %s contains invalid WAVE format\n", args[0])
		os.Exit(1)
	}

	var chunkId [4]byte
	if _, err := io.ReadFull(reader, chunkId[:]); err != nil || string(chunkId[:]) != "fmt " {
		fmt.Fprintf(os.Stderr, "File %s misses fmt chunk\n", args[0])
		os.Exit(1)
	}

	var fmtChunkSize uint32
	if err = binary.Read(reader, binary.LittleEndian, &fmtChunkSize); err != nil {
		fmt.Fprintf(os.Stderr, "File %s misses fmt chunk size\n", args[0])
		os.Exit(1)
	}

	var audioFormat uint16
	if err = binary.Read(reader, binary.LittleEndian, &audioFormat); err != nil || audioFormat != 1 {
		fmt.Fprintf(os.Stderr, "File %s contains unsupported audio format\n", args[0])
		os.Exit(1)
	}

	var numChannels uint16
	if err = binary.Read(reader, binary.LittleEndian, &numChannels); err != nil || (numChannels != 1 && numChannels != 2) {
		fmt.Fprintf(os.Stderr, "File %s contains unsupported number of channels %d\n", args[0], numChannels)
		os.Exit(1)
	}

	var sampleRate uint32
	if err = binary.Read(reader, binary.LittleEndian, &sampleRate); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading sample rate from %s\n", args[0])
		os.Exit(1)
	}
	fmt.Printf("Sample rate: %d\n", sampleRate)

	var byteRate uint32
	if err = binary.Read(reader, binary.LittleEndian, &byteRate); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading sample byte from %s\n", args[0])
		os.Exit(1)
	}
	fmt.Printf("Byte rate: %d\n", byteRate)

	var blockAlign uint16
	if err = binary.Read(reader, binary.LittleEndian, &blockAlign); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading block align from %s\n", args[0])
		os.Exit(1)
	}
	fmt.Printf("Block align: %d\n", blockAlign)

	var bitsPerSample uint16
	if err = binary.Read(reader, binary.LittleEndian, &bitsPerSample); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading bits per sample from %s\n", args[0])
		os.Exit(1)
	}
	fmt.Printf("Bits per sample: %d\n", bitsPerSample)

	if byteRate != sampleRate*uint32(numChannels)*uint32(bitsPerSample)/8 {
		fmt.Fprintf(os.Stderr, "File %s contains malformed byte rate\n", args[0])
		os.Exit(1)
	}

	if blockAlign != numChannels*bitsPerSample/8 {
		fmt.Fprintf(os.Stderr, "File %s contains malformed block align\n", args[0])
		os.Exit(1)
	}

	// Skip remaining fmt chunk bytes if any
	if fmtChunkSize > 16 {
		_, err := reader.Seek(int64(fmtChunkSize-16), io.SeekCurrent)
		if err != nil {
			return
		}
	}

	var audioData []byte
	var otoFormat oto.Format

	decoder, formatPtr := getDecoder(bitsPerSample)
	if formatPtr == nil {
		fmt.Fprintf(os.Stderr, "Unsupported bits per sample: %d\n", bitsPerSample)
		os.Exit(1)
	}
	otoFormat = *formatPtr

	for {
		var chunkId [4]byte
		if _, err := io.ReadFull(reader, chunkId[:]); err != nil {
			break
		}
		var chunkSize uint32
		if err := binary.Read(reader, binary.LittleEndian, &chunkSize); err != nil {
			break
		}

		if string(chunkId[:]) == "data" {
			rawAudioData := make([]byte, chunkSize)
			if _, err := io.ReadFull(reader, rawAudioData); err != nil {
				// Handle unexpected EOF if necessary, but here we just take what we have
			}
			audioData = decoder(rawAudioData)
			break
		}
		_, err := reader.Seek(int64(chunkSize), io.SeekCurrent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error seeking chunk: %d\n", bitsPerSample)
			os.Exit(1)
		}
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
