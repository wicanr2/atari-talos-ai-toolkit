package st

import "fmt"

const rawFloppySectorSize = 512

// RawFloppy is an immutable sector-by-sector Atari ST floppy image.
// Geometry comes from the GEMDOS boot-sector BPB; low-level protected layouts
// require a richer format and are deliberately rejected by this type.
type RawFloppy struct {
	data            []byte
	tracks          uint16
	sides           uint16
	sectorsPerTrack uint16
}

func NewRawFloppy(image []byte) (*RawFloppy, error) {
	if len(image) < rawFloppySectorSize {
		return nil, fmt.Errorf("st: raw floppy is %d bytes, smaller than one sector", len(image))
	}
	bytesPerSector := littleEndianWord(image[0x0b:0x0d])
	totalSectors := littleEndianWord(image[0x13:0x15])
	sectorsPerTrack := littleEndianWord(image[0x18:0x1a])
	sides := littleEndianWord(image[0x1a:0x1c])
	if bytesPerSector != rawFloppySectorSize {
		return nil, fmt.Errorf("st: raw floppy sector size %d, want %d", bytesPerSector, rawFloppySectorSize)
	}
	if totalSectors == 0 || sectorsPerTrack == 0 || (sides != 1 && sides != 2) {
		return nil, fmt.Errorf("st: invalid raw floppy geometry: sectors=%d sectors/track=%d sides=%d",
			totalSectors, sectorsPerTrack, sides)
	}
	wantBytes := uint64(totalSectors) * uint64(bytesPerSector)
	if wantBytes != uint64(len(image)) {
		return nil, fmt.Errorf("st: raw floppy length %d, BPB requires %d", len(image), wantBytes)
	}
	sectorsPerCylinder := uint32(sectorsPerTrack) * uint32(sides)
	if uint32(totalSectors)%sectorsPerCylinder != 0 {
		return nil, fmt.Errorf("st: raw floppy sectors %d not divisible by cylinder size %d",
			totalSectors, sectorsPerCylinder)
	}
	tracks := uint16(uint32(totalSectors) / sectorsPerCylinder)
	if tracks == 0 {
		return nil, fmt.Errorf("st: raw floppy has zero tracks")
	}
	return &RawFloppy{
		data:            append([]byte(nil), image...),
		tracks:          tracks,
		sides:           sides,
		sectorsPerTrack: sectorsPerTrack,
	}, nil
}

func (f *RawFloppy) Geometry() (tracks, sides, sectorsPerTrack uint16) {
	return f.tracks, f.sides, f.sectorsPerTrack
}

func (f *RawFloppy) Sector(track, side, sector uint16) ([]byte, error) {
	if track >= f.tracks || side >= f.sides || sector == 0 || sector > f.sectorsPerTrack {
		return nil, fmt.Errorf("st: raw floppy CHS out of range: track=%d side=%d sector=%d", track, side, sector)
	}
	linear := (uint32(track)*uint32(f.sides)+uint32(side))*uint32(f.sectorsPerTrack) + uint32(sector-1)
	offset := linear * rawFloppySectorSize
	return append([]byte(nil), f.data[offset:offset+rawFloppySectorSize]...), nil
}

func littleEndianWord(value []byte) uint16 {
	return uint16(value[0]) | uint16(value[1])<<8
}
