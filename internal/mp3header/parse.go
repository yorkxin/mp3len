package mp3header

// For MPEG header format, see: http://www.mp3-tech.org/programmer/frame_header.html

import (
	"fmt"
)

const (
	mpegFlagFrameSync     = 0b11111111_11100000_00000000_00000000
	mpegFlagAudioVersion  = 0b00000000_00011000_00000000_00000000
	mpegFlagLayerDesc     = 0b00000000_00000110_00000000_00000000
	mpegFlagProtectionBit = 0b00000000_00000001_00000000_00000000
	mpegFlagBitRate       = 0b00000000_00000000_11110000_00000000
	mpegFlagSampleFreq    = 0b00000000_00000000_00001100_00000000
	mpegFlagPaddingBit    = 0b00000000_00000000_00000010_00000000
	mpegFlagPrivateBit    = 0b00000000_00000000_00000001_00000000
	mpegFlagChannelMode   = 0b00000000_00000000_00000000_11000000
	mpegFlagModeExtension = 0b00000000_00000000_00000000_00110000
	mpegFlagCopyright     = 0b00000000_00000000_00000000_00001000
	mpegFlagOriginal      = 0b00000000_00000000_00000000_00000100
	mpegFLagEmphasis      = 0b00000000_00000000_00000000_00000011
)

const (
	Version2_5     = 0b00
	VersionInvalid = 0b01
	Version2       = 0b10
	Version1       = 0b11
)

const (
	LayerInvalid = 0b00
	Layer3       = 0b01
	Layer2       = 0b10
	Layer1       = 0b11
)

const (
	ChannelModeStereo      = 0b00
	ChannelModeJointStereo = 0b01
	ChannelModeDualMono    = 0b10
	ChannelModeMono        = 0b11
)

type AudioVersionValue uint32
type LayerValue uint32

type MP3Header struct {
	AudioVersion AudioVersionValue
	Layer        LayerValue
	BitRate      int
	SampleFreq   int
	ChannelMode  int
}

func (h MP3Header) String() string {
	return fmt.Sprintf(
		"MPEG-%s Layer %s, %d kbps, %dHz",
		[]string{"2.5", "?", "2", "1"}[h.AudioVersion],
		[]string{"?", "III", "II", "I"}[int(h.Layer)],
		h.BitRate,
		h.SampleFreq,
	)
}

type bitRateArray [16]int
type bitRateLayerDict map[LayerValue]bitRateArray

// 0 means free format
// -1 means bad bit rate
var bitRateTopDict = map[AudioVersionValue]bitRateLayerDict{
	Version1: {
		Layer1: [16]int{0, 32, 64, 92, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, -1},
		Layer2: [16]int{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, -1},
		Layer3: [16]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 176, 192, 224, 256, -1},
	},
	// Version2 is for Version2_5 too
	Version2: {
		Layer1: [16]int{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, -1},
		Layer2: [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, -1},
		Layer3: [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, -1},
	},
}

var sampleRateDict = map[AudioVersionValue][4]int{
	Version1:   {44100, 48000, 32000, -1},
	Version2:   {22050, 24000, 16000, -1},
	Version2_5: {11025, 12000, 8000, -1},
}

func getBitRate(version AudioVersionValue, layer LayerValue, bitRateIndex int) (int, error) {
	if version == Version2_5 {
		version = Version2
	}

	layerDict, ok := bitRateTopDict[version]
	if !ok {
		return -1, fmt.Errorf("Invalid version: %02b", version)
	}

	bitRateLookup, ok := layerDict[layer]

	if !ok {
		return -1, fmt.Errorf("Invalid layer: %02b", layer)
	}

	if bitRateIndex > len(bitRateLookup) {
		return -1, fmt.Errorf("Invalid bitRateIndex: %04b", bitRateIndex)
	}

	return bitRateLookup[bitRateIndex], nil
}

func getSampleFreq(version AudioVersionValue, sampleRateIndex int) (int, error) {
	sampleRateLookup, ok := sampleRateDict[version]

	if !ok {
		return -1, fmt.Errorf("Invalid version: %02b", version)
	}

	if sampleRateIndex > len(sampleRateLookup) {
		return -1, fmt.Errorf("Invalid sampleRateIndex: %04b", sampleRateIndex)
	}

	return sampleRateLookup[sampleRateIndex], nil
}

// ParseMP3Header parses MP3 header by reading a 4-byte data.
func ParseMP3Header(headerBits uint32) (header MP3Header, err error) {
	header = MP3Header{
		AudioVersion: 0xF,
		Layer:        0xF,
		BitRate:      -1,
		SampleFreq:   -1,
	}

	if headerBits&mpegFlagFrameSync == 0 {
		err = fmt.Errorf("MP3 frame sync not found (expecting %X, but found %X)", mpegFlagFrameSync, headerBits)
		return
	}

	header.AudioVersion = AudioVersionValue((headerBits & mpegFlagAudioVersion) >> 19)
	header.Layer = LayerValue((headerBits & mpegFlagLayerDesc) >> 17)
	bitRateIndex := int((headerBits & mpegFlagBitRate) >> 12)
	sampleFreqIndex := int((headerBits & mpegFlagSampleFreq) >> 10)
	header.ChannelMode = int((headerBits & mpegFlagChannelMode) >> 6)

	bitRate, err := getBitRate(header.AudioVersion, header.Layer, bitRateIndex)

	if err != nil {
		return
	}

	header.BitRate = bitRate

	sampleFreq, err := getSampleFreq(header.AudioVersion, sampleFreqIndex)

	if err != nil {
		return
	}

	header.SampleFreq = sampleFreq

	return
}
