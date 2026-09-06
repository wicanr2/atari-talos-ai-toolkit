package st

const floppyMediaReceiptCapacity = 8

type floppyMediaPhase uint8

const (
	floppyMediaIdle floppyMediaPhase = iota
	floppyMediaTrackSelector
	floppyMediaTrackData
	floppyMediaDriveSelector
	floppyMediaDriveRead
	floppyMediaDriveWrite
	floppyMediaSectorSelector
	floppyMediaSectorData
	floppyMediaAddressLow
	floppyMediaAddressMiddle
	floppyMediaAddressHigh
	floppyMediaDMAResetRead
	floppyMediaDMAResetWrite
	floppyMediaCount
	floppyMediaCommandSelector
	floppyMediaReadBusy
	floppyMediaReadTransfer
	floppyMediaReadIRQReset
	floppyMediaReadDMAStatus
	floppyMediaReadIRQSelector
	floppyMediaReadStatusRead
	floppyMediaPostRead
	floppyMediaTimeoutSelector
	floppyMediaForceInterrupt
	floppyMediaSeekDataSelector
	floppyMediaSeekData
	floppyMediaSeekCommandSelector
	floppyMediaSeekBusy
	floppyMediaSeekIRQ
	floppyMediaStatusSelector
	floppyMediaStatusRead
)

type floppyMediaReceipt struct {
	Attempt uint64
	// LockedTrack marks the pass that locked the track. Only the first call to
	// flop_mediach() does it, and it is what makes that pass two transactions
	// longer than every pass after it.
	LockedTrack          bool
	Track                byte
	Drive                int8
	Side                 uint8
	TrackWriteClock      uint64
	DrivePort            byte
	DriveWriteClock      uint64
	Sector               uint16
	DMAAddressStage      uint8
	DMAResetCount        uint8
	ReadCommand          byte
	ReadCommandClock     uint64
	ReadCompleteClock    uint64
	SectorsRead          uint16
	BytesRead            uint32
	TimeoutSelectorClock uint64
	ForceInterrupt       byte
	ForceInterruptClock  uint64
	SeekData             byte
	SeekCommand          byte
	SeekStartClock       uint64
	InactivePolls        uint8
	IRQObserved          bool
	StatusReadClock      uint64
}

type floppyMediaReceipts struct {
	Total   uint64
	Count   uint8
	Next    uint8
	Entries [floppyMediaReceiptCapacity]floppyMediaReceipt
}

func (r *floppyMediaReceipts) append(receipt floppyMediaReceipt) {
	r.Total++
	receipt.Attempt = r.Total
	r.Entries[r.Next] = receipt
	r.Next = (r.Next + 1) % floppyMediaReceiptCapacity
	if r.Count < floppyMediaReceiptCapacity {
		r.Count++
	}
}

func (r *floppyMediaReceipts) attempt(attempt uint64) (floppyMediaReceipt, bool) {
	if attempt == 0 || attempt > r.Total || r.Total-attempt >= uint64(r.Count) {
		return floppyMediaReceipt{}, false
	}
	oldest := (int(r.Next) - int(r.Count) + floppyMediaReceiptCapacity) % floppyMediaReceiptCapacity
	index := (oldest + int(attempt-(r.Total-uint64(r.Count)+1))) % floppyMediaReceiptCapacity
	receipt := r.Entries[index]
	return receipt, receipt.Attempt == attempt
}

func (r *floppyMediaReceipts) reset() {
	*r = floppyMediaReceipts{}
}
