package st

import "testing"

func TestFloppyMediaReceiptsWrapAndReset(t *testing.T) {
	var receipts floppyMediaReceipts
	for attempt := uint64(1); attempt <= 12; attempt++ {
		receipts.append(floppyMediaReceipt{ReadCommandClock: attempt * 100})
	}
	if receipts.Total != 12 || receipts.Count != floppyMediaReceiptCapacity || receipts.Next != 4 {
		t.Fatalf("ring total/count/next=%d/%d/%d", receipts.Total, receipts.Count, receipts.Next)
	}
	for attempt := uint64(1); attempt <= 12; attempt++ {
		receipt, ok := receipts.attempt(attempt)
		wantOK := attempt >= 5
		if ok != wantOK {
			t.Fatalf("attempt %d present=%v want %v", attempt, ok, wantOK)
		}
		if ok && (receipt.Attempt != attempt || receipt.ReadCommandClock != attempt*100) {
			t.Fatalf("attempt %d receipt=%+v", attempt, receipt)
		}
	}
	receipts.reset()
	if receipts != (floppyMediaReceipts{}) {
		t.Fatalf("reset retained receipts: %+v", receipts)
	}
}
