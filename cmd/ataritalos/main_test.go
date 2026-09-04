package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestServeJSONLines(t *testing.T) {
	input := strings.NewReader("{\"id\":\"1\",\"op\":\"hello\"}\n" +
		"{\"id\":\"2\",\"op\":\"run_frames\",\"frames\":1}\n" +
		"{\"id\":\"3\",\"op\":\"quit\"}\n" +
		"{\"id\":\"4\",\"op\":\"hello\"}\n")
	var output bytes.Buffer
	if err := serve(input, &output); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{
		`"id":"1","ok":true`,
		`"id":"2","ok":false`,
		`"code":"not_implemented"`,
		`"id":"3","ok":true`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `"id":"4"`) {
		t.Fatalf("server processed input after quit:\n%s", got)
	}
}
