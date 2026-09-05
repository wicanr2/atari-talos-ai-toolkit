# Atari Talos AI Toolkit

**A headless Atari ST/STF parity tool for LLMs and automated tests.**

Atari Talos exists to help large language models run, observe, and compare Atari ST games more
reliably, producing deterministic input, frame stepping, state capture, and machine-readable
evidence for more accurate retro-game remakes.

The project is at M2. The versioned JSON Lines control protocol and CLI exist; the MC68000 core
passes 232,500 external corpus cases, and the incremental ST/STF machine core executes 7,599
instructions of a fixed EmuTOS 1.3 ROM before correctly entering the MC68000 stopped state. The
MFP USART reset-write boundary is in parity with Hatari; interrupt wake-up and peripheral IRQs are
the next boot gate. The fixed color profile now samples MFP GPIP as `$A1`, matching EmuTOS monitor
detection; the remaining D3 difference at STOP is the not-yet-produced VBL `frclock` value.
Video, input, disk, and a complete TOS boot are not
implemented yet, so this is not yet a game-capable emulator. Unsupported operations fail closed.
See [README.md](README.md) for the authoritative Traditional Chinese documentation.

Atari Talos is independently rewritten in Go from public hardware specifications. Hatari is an
external compatibility oracle, not a source-code dependency. TOS ROMs and game media are not
distributed.

Licensed under [RRSAL-1.0](LICENSE), a source-available license.
