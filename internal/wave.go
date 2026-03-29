package wave

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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

func getDecoder(bitsPerSample uint16) decoderFunc {
	switch bitsPerSample {
	case 8:
		return decode8Bit
	case 16:
		return decode16Bit
	case 24:
		return decode24Bit
	default:
		return nil
	}
}

type Parser struct {
	reader    *bytes.Reader
	metadata  *Metadata
	audioData []byte
}

type Metadata struct {
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
}

func (m Metadata) String() string {
	return fmt.Sprintf("AudioFormat: %d,\nNumChannels: %d,\nSampleRate: %d,\nByteRate: %d,\nBlockAlign: %d,\nBitsPerSample: %d\n",
		m.AudioFormat, m.NumChannels, m.SampleRate, m.ByteRate, m.BlockAlign, m.BitsPerSample)
}

func (p *Parser) GetMetadata() *Metadata {
	return p.metadata
}

func (p *Parser) GetAudioData() []byte {
	return p.audioData
}

func (p *Parser) Parse() error {
	_, err := p.validateHeader()
	if err != nil {
		return err
	}
	metadata, err := p.parseFmt()
	if err != nil {
		return err
	}
	p.metadata = metadata
	audioData, err := p.parseAudioData()
	if err != nil {
		return err
	}
	p.audioData = audioData
	return nil
}

func (p *Parser) validateHeader() (uint32, error) {
	var riff [4]byte
	if _, err := io.ReadFull(p.reader, riff[:]); err != nil || string(riff[:]) != "RIFF" {
		return 0, fmt.Errorf("invalid RIFF header")
	}
	var fileSize uint32
	if err := binary.Read(p.reader, binary.LittleEndian, &fileSize); err != nil {
		return 0, fmt.Errorf("invalid file size")
	}
	var waveFormat [4]byte
	if _, err := io.ReadFull(p.reader, waveFormat[:]); err != nil || string(waveFormat[:]) != "WAVE" {
		return 0, fmt.Errorf("invalid WAVE format")
	}
	return fileSize, nil
}

func (p *Parser) parseFmt() (*Metadata, error) {
	var chunkId [4]byte
	if _, err := io.ReadFull(p.reader, chunkId[:]); err != nil || string(chunkId[:]) != "fmt " {
		return nil, fmt.Errorf("missing fmt chunk")
	}
	var fmtChunkSize uint32
	if err := binary.Read(p.reader, binary.LittleEndian, &fmtChunkSize); err != nil {
		return nil, fmt.Errorf("missing fmt chunk size")
	}
	var audioFormat uint16
	if err := binary.Read(p.reader, binary.LittleEndian, &audioFormat); err != nil || audioFormat != 1 {
		return nil, fmt.Errorf("unsupported audio format")
	}
	var numChannels uint16
	if err := binary.Read(p.reader, binary.LittleEndian, &numChannels); err != nil || (numChannels != 1 && numChannels != 2) {
		return nil, fmt.Errorf("unsupported number of channels")
	}
	var sampleRate uint32
	if err := binary.Read(p.reader, binary.LittleEndian, &sampleRate); err != nil {
		return nil, fmt.Errorf("error reading sample rate")
	}
	var byteRate uint32
	if err := binary.Read(p.reader, binary.LittleEndian, &byteRate); err != nil {
		return nil, fmt.Errorf("error reading sample byte")
	}
	var blockAlign uint16
	if err := binary.Read(p.reader, binary.LittleEndian, &blockAlign); err != nil {
		return nil, fmt.Errorf("error reading block align")
	}
	var bitsPerSample uint16
	if err := binary.Read(p.reader, binary.LittleEndian, &bitsPerSample); err != nil {
		return nil, fmt.Errorf("error reading bits per sample")
	}
	if bitsPerSample != 8 && bitsPerSample != 16 && bitsPerSample != 24 {
		return nil, fmt.Errorf("unsupported bits per sample")
	}

	if byteRate != sampleRate*uint32(numChannels)*uint32(bitsPerSample)/8 {
		return nil, fmt.Errorf("malformed byte rate")
	}

	if blockAlign != numChannels*bitsPerSample/8 {
		return nil, fmt.Errorf("malformed block align")
	}

	// Skip remaining fmt chunk bytes if any
	if fmtChunkSize > 16 {
		_, err := p.reader.Seek(int64(fmtChunkSize-16), io.SeekCurrent)
		if err != nil {
			return nil, fmt.Errorf("error skipping remaining fmt chunk")
		}
	}

	return &Metadata{
		AudioFormat:   audioFormat,
		NumChannels:   numChannels,
		SampleRate:    sampleRate,
		ByteRate:      byteRate,
		BlockAlign:    blockAlign,
		BitsPerSample: bitsPerSample,
	}, nil
}

func (p *Parser) parseAudioData() ([]byte, error) {
	decoder := getDecoder(p.metadata.BitsPerSample)

	var audioData []byte

	for {
		var chunkId [4]byte
		if _, err := io.ReadFull(p.reader, chunkId[:]); err != nil {
			return nil, fmt.Errorf("error reading chunk ID")
		}
		var chunkSize uint32
		if err := binary.Read(p.reader, binary.LittleEndian, &chunkSize); err != nil {
			return nil, fmt.Errorf("error reading chunk size")
		}

		if string(chunkId[:]) == "data" {
			rawAudioData := make([]byte, chunkSize)
			if _, err := io.ReadFull(p.reader, rawAudioData); err != nil {
				// Handle unexpected EOF if necessary, but here we just take what we have
			}
			audioData = decoder(rawAudioData)
			break
		}
		_, err := p.reader.Seek(int64(chunkSize), io.SeekCurrent)
		if err != nil {
			return nil, fmt.Errorf("Error seeking chunk")
		}
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("data chunks not found")
	}

	return audioData, nil
}

func NewParser(data []byte) *Parser {
	return &Parser{reader: bytes.NewReader(data)}
}
