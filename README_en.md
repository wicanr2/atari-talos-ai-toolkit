# Atari Talos AI Toolkit

**A headless Atari ST/STF parity tool for LLMs and automated tests.**

Atari Talos exists to help large language models run, observe, and compare Atari ST games more
reliably, producing deterministic input, frame stepping, state capture, and machine-readable
evidence for more accurate retro-game remakes.

The project is at M2. The versioned JSON Lines control protocol and CLI exist; the MC68000 core
passes 240,000 external corpus cases. The incremental ST/STF core now schedules recurring reset-mode 60 Hz
GLUE VBL events, advances a stopped CPU to the next event, performs E-clock-aligned video IACK,
and executes the real EmuTOS handler. Its second handler entry matches Hatari at 267,332 machine
clocks and increments `$466 frclock` from 1 to 2; a bounded run also crosses the third VBL and
completes the `$FF8260` low-resolution initialization. EmuTOS then changes `$FF820A` from 60 Hz to
50 Hz during line zero; Talos adjusts the fourth VBL deadline to Hatari's 535,528 clocks. The next
EmuTOS's complete 16-color `$FF8240–$FF825E` palette loop now also matches Hatari. The next boot
gate is the `$FF8201` framebuffer-base high-byte write.
EmuTOS 1.3 now boots all the way to the GEM desktop, with the screen contents pinned by a
SHA-256 in the test suite, and injected relative-mouse packets move the on-screen pointer by
the exact delta — verified against EmuTOS's own VDI. Keyboard input, mouse button semantics and
mounting a disk to load a program are not implemented yet, so this is not yet a game-capable
emulator. Unsupported operations fail closed.
See [README.md](README.md) for the authoritative Traditional Chinese documentation.

Atari Talos is independently rewritten in Go from public hardware specifications. Hatari is an
external compatibility oracle, not a source-code dependency. TOS ROMs and game media are not
distributed.

Licensed under [RRSAL-1.0](LICENSE), a source-available license.
