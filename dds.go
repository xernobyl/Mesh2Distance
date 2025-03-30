package main

import (
	"encoding/binary"
	"os"
)

// DDSHeader represents the DDS file header
type DDSHeader struct {
	Magic            uint32
	Size             uint32
	Flags            uint32
	Height           uint32
	Width            uint32
	PitchOrLinear    uint32
	Depth            uint32
	MipMapCount      uint32
	Reserved1        [11]uint32
	PixelFormatSize  uint32
	PixelFormatFlags uint32
	FourCC           uint32
	RGBBitCount      uint32
	RBitMask         uint32
	GBitMask         uint32
	BBitMask         uint32
	ABitMask         uint32
	Caps             uint32
	Caps2            uint32
	Caps3            uint32
	Caps4            uint32
	Reserved2        uint32
}

const (
	DDS_MAGIC        = 0x20534444 // "DDS "
	DDSD_CAPS        = 0x00000001
	DDSD_HEIGHT      = 0x00000002
	DDSD_WIDTH       = 0x00000004
	DDSD_PITCH       = 0x00000008
	DDSD_PIXELFORMAT = 0x00001000
	DDSD_DEPTH       = 0x00800000
	DDSD_MIPMAPCOUNT = 0x00020000
	DDPF_RGB         = 0x00000040
	DDPF_ALPHAPIXELS = 0x00000001
	DDSCAPS_TEXTURE  = 0x00001000
	DDSCAPS_COMPLEX  = 0x00000008
	DDSCAPS_MIPMAP   = 0x00400000
	DDSCAPS2_VOLUME  = 0x00200000
)

// Save3DTextureAsDDS saves a 3D texture as a DDS file
func Save3DTextureAsDDS(filename string, data []byte, width, height, depth int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	header := DDSHeader{
		Magic:            DDS_MAGIC,
		Size:             124,
		Flags:            DDSD_CAPS | DDSD_HEIGHT | DDSD_WIDTH | DDSD_PIXELFORMAT | DDSD_DEPTH,
		Height:           uint32(height),
		Width:            uint32(width),
		Depth:            uint32(depth),
		PitchOrLinear:    uint32(len(data) / depth),
		PixelFormatSize:  8,
		PixelFormatFlags: DDPF_ALPHAPIXELS,
		RGBBitCount:      8, // Assuming RGB format
		RBitMask:         0x00FF0000,
		GBitMask:         0x0000FF00,
		BBitMask:         0x000000FF,
		Caps:             DDSCAPS_TEXTURE | DDSCAPS_COMPLEX,
		Caps2:            DDSCAPS2_VOLUME,
	}

	// Write header
	if err := binary.Write(file, binary.LittleEndian, &header); err != nil {
		return err
	}

	// Write texture data
	if _, err := file.Write(data); err != nil {
		return err
	}

	return nil
}
