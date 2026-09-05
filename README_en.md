# Atari Talos AI Toolkit

**A headless Atari ST/STF parity tool for LLMs and automated tests.**

Atari Talos exists to help large language models run, observe, and compare Atari ST games more
reliably, producing deterministic input, frame stepping, state capture, and machine-readable
evidence for more accurate retro-game remakes.

The project is at M2. The versioned JSON Lines control protocol and CLI exist; the MC68000 core
passes 232,500 external corpus cases. The incremental ST/STF core now schedules recurring 50 Hz
GLUE VBL events, advances a stopped CPU to the next event, performs E-clock-aligned video IACK,
and executes the real EmuTOS handler. Its second handler entry matches Hatari at 293,984 machine
clocks and increments `$466 frclock` from 1 to 2; a bounded run also crosses the third VBL. The next
boot gate is the `$FF8260` Shifter resolution write.
Video, input, disk, and a complete TOS boot are not
implemented yet, so this is not yet a game-capable emulator. Unsupported operations fail closed.
See [README.md](README.md) for the authoritative Traditional Chinese documentation.

Atari Talos is independently rewritten in Go from public hardware specifications. Hatari is an
external compatibility oracle, not a source-code dependency. TOS ROMs and game media are not
distributed.

Licensed under [RRSAL-1.0](LICENSE), a source-available license.
