package protocol

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/st"
)

// maxRunInstructions 是一次 run_instructions 的上限。開機到桌面大約
// 一千四百萬條，留一個數量級的餘裕。
const maxRunInstructions = 100_000_000

// Session 把一台機器接到 talos-jsonl 契約上（規格 147）。ROM 由行程啟動時
// 決定，不由請求指定——請求裡不接受檔案路徑。
type Session struct {
	romPath string
	machine *st.Machine
}

// NewSession 讀 TALOS_TOS_ROM 決定 ROM 的位置，此時還不開機。
func NewSession() *Session {
	return &Session{romPath: os.Getenv("TALOS_TOS_ROM")}
}

// Handle 先看本層實作的 op，其餘交給無狀態的 Handle。
func (s *Session) Handle(request Request) (Response, bool) {
	switch request.Op {
	case "capabilities":
		return success(request.ID, map[string]any{
			"commands": []string{"hello", "capabilities", "boot", "reset",
				"run_instructions", "key", "mouse", "framebuffer", "quit"},
			"machine": "atari-stf", "emulation_ready": true,
		}), false
	case "boot":
		return s.boot(request), false
	case "reset":
		s.machine = nil
		return success(request.ID, map[string]any{"booted": false}), false
	case "run_instructions":
		return s.runInstructions(request), false
	case "mouse":
		return s.mouse(request), false
	case "key":
		return s.key(request), false
	case "framebuffer":
		return s.framebuffer(request), false
	}
	return Handle(request)
}

func (s *Session) boot(request Request) Response {
	if s.machine != nil {
		return failure(request.ID, "already_booted", "the machine is already booted; reset first")
	}
	if s.romPath == "" {
		return failure(request.ID, "no_rom", "TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(s.romPath)
	if err != nil {
		return failure(request.ID, "no_rom", err.Error())
	}
	machine, err := st.NewMachine(st.RAM1M, rom)
	if err != nil {
		return failure(request.ID, "boot_failed", err.Error())
	}
	if err := machine.Reset(); err != nil {
		return failure(request.ID, "boot_failed", err.Error())
	}
	s.machine = machine
	return success(request.ID, map[string]any{
		"booted": true, "rom_bytes": len(rom),
		"rom_sha256": fmt.Sprintf("%x", sha256.Sum256(rom)),
	})
}

func (s *Session) runInstructions(request Request) Response {
	if s.machine == nil {
		return notBooted(request.ID)
	}
	if request.Count == 0 || request.Count > maxRunInstructions {
		return failure(request.ID, "invalid_request",
			fmt.Sprintf("count must be 1..%d", maxRunInstructions))
	}
	for step := uint64(0); step < request.Count; step++ {
		if _, err := s.machine.Step(); err != nil {
			return Response{ID: request.ID, OK: false,
				Error: &Error{Code: "bus_fault", Message: err.Error()},
			}
		}
	}
	return success(request.ID, s.state())
}

func (s *Session) mouse(request Request) Response {
	if s.machine == nil {
		return notBooted(request.ID)
	}
	if request.DX < -128 || request.DX > 127 || request.DY < -128 || request.DY > 127 {
		return failure(request.ID, "invalid_request", "dx and dy must be -128..127")
	}
	if err := s.machine.QueueMouseMotion(request.DX, request.DY, request.Left, request.Right); err != nil {
		return failure(request.ID, "unsupported_input", err.Error())
	}
	return success(request.ID, s.state())
}

func (s *Session) key(request Request) Response {
	if s.machine == nil {
		return notBooted(request.ID)
	}
	if request.ScanCode == 0 || request.ScanCode > 0xff {
		return failure(request.ID, "invalid_request", "scan_code must be 1..255")
	}
	if err := s.machine.QueueKey(byte(request.ScanCode), request.Pressed); err != nil {
		return failure(request.ID, "unsupported_input", err.Error())
	}
	return success(request.ID, s.state())
}

func (s *Session) framebuffer(request Request) Response {
	if s.machine == nil {
		return notBooted(request.ID)
	}
	frame, base, resolution, err := s.machine.Framebuffer()
	if err != nil {
		return failure(request.ID, "bus_fault", err.Error())
	}
	return success(request.ID, map[string]any{
		"base": base, "resolution": resolution, "bytes": len(frame),
		"sha256": fmt.Sprintf("%x", sha256.Sum256(frame)),
	})
}

func (s *Session) state() map[string]any {
	return map[string]any{
		"instructions": s.machine.Instructions,
		"interrupts":   s.machine.Interrupts,
		"clocks":       s.machine.Clocks,
	}
}

func notBooted(id string) Response {
	return failure(id, "not_booted", "the machine is not booted; send boot first")
}
