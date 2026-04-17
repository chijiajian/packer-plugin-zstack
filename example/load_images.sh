#!/usr/bin/env sh
set -eu

echo "[load_images] start"
echo "[load_images] user: $(id -un)"
echo "[load_images] hostname: $(hostname)"
echo "[load_images] kernel: $(uname -a)"
date

if command -v docker >/dev/null 2>&1; then
  echo "[load_images] docker found, printing version"
  docker --version || true
else
  echo "[load_images] docker not found, skip docker checks"
fi

echo "[load_images] done"
