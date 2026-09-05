package st

import "fmt"

const (
	LowResolutionWidth       = 320
	LowResolutionHeight      = 200
	lowResolutionLineBytes   = 160
	lowResolutionFrameBytes  = lowResolutionLineBytes * LowResolutionHeight
	lowResolutionPixelCount  = LowResolutionWidth * LowResolutionHeight
	lowResolutionGroupBytes  = 8
	lowResolutionGroupPixels = 16
)

type IndexedFrame struct {
	Width   int
	Height  int
	Indices []byte
	Palette [16]uint16
}

func DecodeLowResolutionPlanar(source []byte) ([]byte, error) {
	if len(source) != lowResolutionFrameBytes {
		return nil, fmt.Errorf("st: low-resolution framebuffer size %d, want %d",
			len(source), lowResolutionFrameBytes)
	}
	indices := make([]byte, lowResolutionPixelCount)
	for sourceOffset, pixelOffset := 0, 0; sourceOffset < len(source); sourceOffset, pixelOffset = sourceOffset+lowResolutionGroupBytes, pixelOffset+lowResolutionGroupPixels {
		var planes [4]uint16
		for plane := range planes {
			offset := sourceOffset + plane*2
			planes[plane] = uint16(source[offset])<<8 | uint16(source[offset+1])
		}
		for pixel := 0; pixel < lowResolutionGroupPixels; pixel++ {
			mask := uint16(0x8000 >> pixel)
			var index byte
			for plane, word := range planes {
				if word&mask != 0 {
					index |= 1 << plane
				}
			}
			indices[pixelOffset+pixel] = index
		}
	}
	return indices, nil
}

func (m *Memory) LowResolutionFrame() (IndexedFrame, error) {
	if m.shifterResolution != 0 {
		return IndexedFrame{}, m.fault(ShifterResolution, 5, false, 1, FaultUnsupportedDeviceState)
	}
	source := make([]byte, lowResolutionFrameBytes)
	base := m.ActiveVideoBase()
	if base+lowResolutionFrameBytes > 0x0040_0000 {
		return IndexedFrame{}, fmt.Errorf("st: framebuffer DMA window 0x%06x..0x%06x exceeds 22-bit range",
			base, base+lowResolutionFrameBytes-1)
	}
	for offset := range source {
		address := base + uint32(offset)
		physical, ok := m.ramAddress(address)
		if !ok {
			return IndexedFrame{}, fmt.Errorf("st: framebuffer DMA address 0x%06x is not mapped RAM", address)
		}
		source[offset] = m.ram[physical]
	}
	indices, err := DecodeLowResolutionPlanar(source)
	if err != nil {
		return IndexedFrame{}, err
	}
	return IndexedFrame{
		Width:   LowResolutionWidth,
		Height:  LowResolutionHeight,
		Indices: indices,
		Palette: m.shifterPalette,
	}, nil
}
