// SPDX-License-Identifier: GPL-3.0-or-later
// Package goblin exposes embedded documentation for the goblin binary.
package goblin

import _ "embed"

// Guide is the full contents of USING.md, embedded at build time.
// It is printed by goblin -guide and is intended for AI agents that
// need strategy guidance without repository access.
//
//go:embed USING.md
var Guide string
