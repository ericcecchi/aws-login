#!/usr/bin/env bash
set -euo pipefail

version="${1:-}"
if [[ -z "$version" ]]; then
  echo "usage: $0 <version>" >&2
  exit 1
fi

file="constants.go"

python3 - "$file" "$version" <<'PY'
import re
import sys

path = sys.argv[1]
version = sys.argv[2]

with open(path, 'r', encoding='utf-8') as f:
    data = f.read()

pattern = re.compile(r'(?m)^(?P<prefix>\s*version\s*=\s*)"[^"]*"')
new_data, count = pattern.subn(r'\g<prefix>"%s"' % version, data, count=1)
if count != 1:
    raise SystemExit(f"version constant not found in {path}")

with open(path, 'w', encoding='utf-8') as f:
    f.write(new_data)
PY
