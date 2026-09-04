# Atari Talos AI Toolkit

**A headless Atari ST/STF parity tool for LLMs and automated tests.**

Atari Talos exists to help large language models run, observe, and compare Atari ST games more
reliably, producing deterministic input, frame stepping, state capture, and machine-readable
evidence for more accurate retro-game remakes.

The project is at M0: the versioned JSON Lines control protocol and CLI exist, but the Atari ST
machine core does not. Unsupported operations fail closed. See [README.md](README.md) for the
authoritative Traditional Chinese documentation.

Atari Talos is independently rewritten in Go from public hardware specifications. Hatari is an
external compatibility oracle, not a source-code dependency. TOS ROMs and game media are not
distributed.

Licensed under [RRSAL-1.0](LICENSE), a source-available license.
