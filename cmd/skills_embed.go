package cmd

import "embed"

// EmbeddedSkills contains the agent skills files (SKILL.md + references/)
// baked into the binary at build time.
//
//go:embed skills/healthsync
var EmbeddedSkills embed.FS
