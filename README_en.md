# Atari Talos AI Toolkit

**A headless Atari ST/STF parity tool for LLMs and automated tests.**

Atari Talos exists to help large language models run, observe, and compare Atari ST games more
reliably, producing deterministic input, frame stepping, state capture, and machine-readable
evidence for more accurate retro-game remakes.

The project is at M2. The versioned JSON Lines control protocol and CLI exist; the MC68000 core
passes 232,500 external corpus cases. The incremental ST/STF core now raises its first GLUE VBL,
accepts the level-4 autovector, and executes the real EmuTOS handler that increments `$466 frclock`
to 1. It then reaches the second VBL wait after 7,604 guest instructions and 178,228 machine clocks.
The fixed color profile samples MFP GPIP as `$A1`, matching EmuTOS monitor detection; recurring VBL
scheduling and stopped-clock advancement are the next boot gate.
Video, input, disk, and a complete TOS boot are not
implemented yet, so this is not yet a game-capable emulator. Unsupported operations fail closed.
See [README.md](README.md) for the authoritative Traditional Chinese documentation.

Atari Talos is independently rewritten in Go from public hardware specifications. Hatari is an
external compatibility oracle, not a source-code dependency. TOS ROMs and game media are not
distributed.

Licensed under [RRSAL-1.0](LICENSE), a source-available license.
