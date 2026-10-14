#!/usr/bin/env bash
# Copyright ZStack.io, Inc. 2013, 2026
# SPDX-License-Identifier: MPL-2.0
set -euo pipefail

compiler="$(cd "$(dirname "$0")" && pwd)/compile-to-webdocs.sh"
task_dir=$(mktemp -d)
trap 'rm -rf "$task_dir"' EXIT
mkdir -p "$task_dir/source docs/builders" "$task_dir/source docs/stories"
cat > "$task_dir/source docs/README.md" <<'DOC'
[ISO](/packer/plugins/builders/zstack/iso)
DOC
cat > "$task_dir/source docs/builders/plain.mdx" <<'DOC'
First content line must remain.

Type: `zstack`
DOC
cat > "$task_dir/source docs/builders/headed.mdx" <<'DOC'
# Builder heading

Type: `zstack-iso`
DOC
cat > "$task_dir/source docs/builders/meta.mdx" <<'DOC'
---
title: Example
---

# Builder heading

Type: `zstack-iso`
DOC
printf 'Local task content\n' > "$task_dir/source docs/stories/private.mdx"
bash "$compiler" "$task_dir" "source docs" "output docs" "chijiajian"
plain="$task_dir/output docs/components/builder/plain/README.md"
headed="$task_dir/output docs/components/builder/headed/README.md"
meta="$task_dir/output docs/components/builder/meta/README.md"
grep -q '^First content line must remain\.$' "$plain"
grep -q '^Type: `zstack`$' "$plain"
for file in "$headed" "$meta"; do
  grep -q '^Type: `zstack-iso`$' "$file"
  ! grep -q '^# Builder heading\|^title:\|^---$' "$file"
done
grep -q '/packer/integrations/chijiajian/zstack/latest/components/builder/iso' "$task_dir/output docs/README.md"
! find "$task_dir/output docs" -type f -exec grep -l 'Local task content' {} + | grep -q .
printf 'Web-docs compiler checks passed\n'
