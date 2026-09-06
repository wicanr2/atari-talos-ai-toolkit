package protocol

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func newTestSession(t *testing.T) *Session {
	t.Helper()
	if os.Getenv("TALOS_TOS_ROM") == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	return NewSession()
}

func do(t *testing.T, session *Session, line string) Response {
	t.Helper()
	request, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("decode %s: %v", line, err)
	}
	response, _ := session.Handle(request)
	return response
}

// 沒開機就送輸入或執行，一律 not_booted（規格 147）。
func TestSessionRefusesInputBeforeBoot(t *testing.T) {
	session := NewSession()
	for _, line := range []string{
		`{"id":"1","op":"run_instructions","count":10}`,
		`{"id":"2","op":"mouse","dx":1}`,
		`{"id":"3","op":"key","scan_code":28,"pressed":true}`,
		`{"id":"4","op":"framebuffer"}`,
	} {
		response := do(t, session, line)
		if response.OK || response.Error.Code != "not_booted" {
			t.Errorf("%s 回的是 %+v", line, response.Error)
		}
	}
}

func TestSessionRejectsOutOfRangeInput(t *testing.T) {
	session := newTestSession(t)
	if response := do(t, session, `{"id":"b","op":"boot"}`); !response.OK {
		t.Fatalf("boot 失敗：%+v", response.Error)
	}
	for _, test := range []struct{ line, code string }{
		{`{"id":"1","op":"run_instructions","count":0}`, "invalid_request"},
		{`{"id":"2","op":"run_instructions","count":100000001}`, "invalid_request"},
		{`{"id":"3","op":"mouse","dx":128}`, "invalid_request"},
		{`{"id":"4","op":"mouse","dy":-129}`, "invalid_request"},
		{`{"id":"5","op":"key","scan_code":0,"pressed":true}`, "invalid_request"},
		{`{"id":"6","op":"key","scan_code":115,"pressed":true}`, "unsupported_input"},
		{`{"id":"7","op":"boot"}`, "already_booted"},
	} {
		response := do(t, session, test.line)
		if response.OK || response.Error.Code != test.code {
			t.Errorf("%s 回的是 %+v，應該是 %s", test.line, response.Error, test.code)
		}
	}
	// 未知欄位一律拒絕。
	if _, err := Decode([]byte(`{"id":"8","op":"mouse","wiggle":1}`)); err == nil {
		t.Error("未知欄位竟然通過")
	}
	// reset 之後又回到沒開機。
	if response := do(t, session, `{"id":"9","op":"reset"}`); !response.OK {
		t.Fatalf("reset 失敗：%+v", response.Error)
	}
	if response := do(t, session, `{"id":"a","op":"framebuffer"}`); response.OK {
		t.Error("reset 之後 framebuffer 竟然成功")
	}
}

func TestSessionCapabilitiesReportEmulationReady(t *testing.T) {
	session := NewSession()
	response := do(t, session, `{"id":"1","op":"capabilities"}`)
	if !response.OK || response.Result["emulation_ready"] != true {
		t.Fatalf("capabilities=%+v", response.Result)
	}
	// 沒實作的還是要照實回報。
	for _, line := range []string{
		`{"id":"2","op":"run_frames","frames":1}`,
		`{"id":"3","op":"snapshot"}`,
		`{"id":"4","op":"trace"}`,
	} {
		if response := do(t, session, line); response.OK ||
			response.Error.Code != "not_implemented" {
			t.Errorf("%s 回的是 %+v", line, response.Error)
		}
	}
}

// 端到端：開機走到桌面，畫面指紋等於規格 140 釘的值；移動滑鼠指紋會變，
// 移回來會復原。
func TestSessionBootsToTheDesktopAndTakesInput(t *testing.T) {
	const desktop = "1de1eb45e862218844abe07ae05fda4c4a9453817ed0ab348a374bca67768f78"
	session := newTestSession(t)
	if response := do(t, session, `{"id":"1","op":"boot"}`); !response.OK {
		t.Fatalf("boot 失敗：%+v", response.Error)
	}
	if response := do(t, session, `{"id":"2","op":"run_instructions","count":14000000}`); !response.OK {
		t.Fatalf("run 失敗：%+v", response.Error)
	}
	response := do(t, session, `{"id":"3","op":"framebuffer"}`)
	if !response.OK || response.Result["sha256"] != desktop {
		t.Fatalf("桌面指紋=%v", response.Result)
	}
	if response.Result["bytes"] != 32000 || response.Result["resolution"] != byte(0) {
		t.Errorf("framebuffer 幾何=%v", response.Result)
	}

	for _, line := range []string{
		`{"id":"4","op":"mouse","dx":30}`,
		`{"id":"5","op":"run_instructions","count":200000}`,
	} {
		if response := do(t, session, line); !response.OK {
			t.Fatalf("%s 失敗：%+v", line, response.Error)
		}
	}
	if response := do(t, session, `{"id":"6","op":"framebuffer"}`); response.Result["sha256"] == desktop {
		t.Error("滑鼠移動之後畫面指紋沒變")
	}
	for _, line := range []string{
		`{"id":"7","op":"mouse","dx":-30}`,
		`{"id":"8","op":"run_instructions","count":200000}`,
	} {
		if response := do(t, session, line); !response.OK {
			t.Fatalf("%s 失敗：%+v", line, response.Error)
		}
	}
	if response := do(t, session, `{"id":"9","op":"framebuffer"}`); response.Result["sha256"] != desktop {
		t.Errorf("移回原點之後指紋=%v", response.Result["sha256"])
	}

	// 鍵盤送得出去（桌面上按 `1` 沒有可見效果，這裡只驗契約這一層）。
	for _, line := range []string{
		`{"id":"a","op":"key","scan_code":2,"pressed":true}`,
		`{"id":"b","op":"run_instructions","count":100000}`,
		`{"id":"c","op":"key","scan_code":2,"pressed":false}`,
		`{"id":"d","op":"run_instructions","count":400000}`,
	} {
		if response := do(t, session, line); !response.OK {
			t.Fatalf("%s 失敗：%+v", line, response.Error)
		}
	}

	// boot 的收據要帶 ROM 的雜湊。
	rom, err := os.ReadFile(os.Getenv("TALOS_TOS_ROM"))
	if err != nil {
		t.Fatal(err)
	}
	session2 := NewSession()
	response = do(t, session2, `{"id":"e","op":"boot"}`)
	if response.Result["rom_sha256"] != fmt.Sprintf("%x", sha256.Sum256(rom)) {
		t.Errorf("boot 回的 ROM 雜湊=%v", response.Result["rom_sha256"])
	}
	if _, err := json.Marshal(response); err != nil {
		t.Fatal(err)
	}
}
