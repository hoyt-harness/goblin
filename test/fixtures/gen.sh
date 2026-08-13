#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Generates a synthetic test fixture: 30s video with 3 scene changes
# and a 440Hz tone. Checked in; output files are gitignored.
# Checked in; output files are gitignored.
# Usage: bash test/fixtures/gen.sh
set -e

OUT="test/fixtures/test.mp4"

ffmpeg -y \
  -f lavfi -i "color=c=red:s=320x240:d=8" \
  -f lavfi -i "color=c=blue:s=320x240:d=8" \
  -f lavfi -i "color=c=green:s=320x240:d=7" \
  -f lavfi -i "color=c=yellow:s=320x240:d=7" \
  -f lavfi -i "sine=frequency=440:duration=30" \
  -filter_complex "[0:v][1:v][2:v][3:v]concat=n=4:v=1:a=0[vout]" \
  -map "[vout]" -map "4:a" \
  -t 30 \
  -c:v libx264 -c:a aac \
  "$OUT"

echo "Generated: $OUT"
