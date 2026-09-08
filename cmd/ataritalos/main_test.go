package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/wicanr2/atari-talos-ai-toolkit/protocol"
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

// serve 這一層要真的把 session 的狀態串起來：同一條連線上 boot 之後，
// 後面的請求看得到那台機器（規格 147）。
func TestServeKeepsTheSessionAcrossRequests(t *testing.T) {
	if os.Getenv("TALOS_TOS_ROM") == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	input := strings.Join([]string{
		`{"id":"1","op":"capabilities"}`,
		`{"id":"2","op":"boot"}`,
		`{"id":"3","op":"run_instructions","count":14000000}`,
		`{"id":"4","op":"framebuffer"}`,
		`{"id":"5","op":"mouse","dx":30}`,
		`{"id":"6","op":"run_instructions","count":200000}`,
		`{"id":"7","op":"framebuffer"}`,
		`{"id":"8","op":"quit"}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	if err := serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 8 {
		t.Fatalf("回了 %d 行，應該是 8 行", len(lines))
	}
	const desktop = "1de1eb45e862218844abe07ae05fda4c4a9453817ed0ab348a374bca67768f78"
	var fingerprints []string
	for index, line := range lines {
		var response protocol.Response
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("第 %d 行不是 JSON：%v", index+1, err)
		}
		if !response.OK {
			t.Fatalf("第 %d 行失敗：%+v", index+1, response.Error)
		}
		if hash, ok := response.Result["sha256"].(string); ok {
			fingerprints = append(fingerprints, hash)
		}
	}
	if len(fingerprints) != 2 || fingerprints[0] != desktop {
		t.Fatalf("畫面指紋=%v", fingerprints)
	}
	if fingerprints[1] == desktop {
		t.Error("滑鼠移動之後指紋沒變")
	}
}
