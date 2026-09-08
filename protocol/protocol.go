package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	Name    = "Atari Talos AI Toolkit"
	Version = "0.0.1-dev"
	Wire    = "talos-jsonl/1"
)

type Request struct {
	ID     string `json:"id"`
	Op     string `json:"op"`
	Frames uint64 `json:"frames,omitempty"`
	// 規格 147：執行與輸入。
	Count    uint64 `json:"count,omitempty"`
	DX       int    `json:"dx,omitempty"`
	DY       int    `json:"dy,omitempty"`
	Left     bool   `json:"left,omitempty"`
	Right    bool   `json:"right,omitempty"`
	ScanCode uint16 `json:"scan_code,omitempty"`
	Pressed  bool   `json:"pressed,omitempty"`
}

type Response struct {
	ID     string         `json:"id,omitempty"`
	OK     bool           `json:"ok"`
	Result map[string]any `json:"result,omitempty"`
	Error  *Error         `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Decode(line []byte) (Request, error) {
	var request Request
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return Request{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Request{}, errors.New("multiple JSON values")
		}
		return Request{}, err
	}
	if request.ID == "" {
		return Request{}, errors.New("id is required")
	}
	if request.Op == "" {
		return Request{}, errors.New("op is required")
	}
	return request, nil
}

func Handle(request Request) (Response, bool) {
	switch request.Op {
	case "hello":
		return success(request.ID, map[string]any{
			"name": Name, "protocol": Wire, "version": Version,
		}), false
	case "capabilities":
		return success(request.ID, map[string]any{
			"commands": []string{"hello", "capabilities", "quit"},
			"machine":  "atari-stf", "emulation_ready": false,
		}), false
	case "quit":
		return success(request.ID, map[string]any{"quitting": true}), true
	case "boot", "reset", "run_instructions", "run_frames", "key", "mouse",
		"read_memory", "write_memory", "breakpoint", "watchpoint", "snapshot",
		"restore", "framebuffer", "trace":
		// 有機器的時候，前六個由 Session 接手（規格 147）；這裡是沒有機器的
		// 無狀態路徑。
		return failure(request.ID, "not_implemented",
			fmt.Sprintf("%s requires the Atari ST machine core", request.Op)), false
	default:
		return failure(request.ID, "unknown_operation", "unknown operation: "+request.Op), false
	}
}

func Invalid(id, message string) Response {
	return failure(id, "invalid_request", message)
}

func success(id string, result map[string]any) Response {
	return Response{ID: id, OK: true, Result: result}
}

func failure(id, code, message string) Response {
	return Response{ID: id, OK: false, Error: &Error{Code: code, Message: message}}
}
