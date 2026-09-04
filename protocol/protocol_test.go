package protocol

import "testing"

func TestHello(t *testing.T) {
	request, err := Decode([]byte(`{"id":"a","op":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	response, quit := Handle(request)
	if quit || !response.OK || response.ID != "a" || response.Result["protocol"] != Wire {
		t.Fatalf("unexpected response: %#v, quit=%v", response, quit)
	}
}

func TestDecodeFailsClosed(t *testing.T) {
	tests := []string{
		`{"id":"a","op":"hello","surprise":true}`,
		`{"id":"","op":"hello"}`,
		`{"id":"a","op":""}`,
		`{"id":"a","op":"hello"} {"id":"b","op":"hello"}`,
	}
	for _, input := range tests {
		if _, err := Decode([]byte(input)); err == nil {
			t.Errorf("Decode(%q) unexpectedly succeeded", input)
		}
	}
}

func TestMachineCommandsAreNotFaked(t *testing.T) {
	response, quit := Handle(Request{ID: "x", Op: "run_frames", Frames: 1})
	if quit || response.OK || response.Error == nil || response.Error.Code != "not_implemented" {
		t.Fatalf("unexpected response: %#v, quit=%v", response, quit)
	}
}
