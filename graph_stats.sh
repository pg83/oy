#!/bin/bash
python3 -c "
import json, sys
with open(sys.argv[1]) as f:
    data = json.load(f)
by_kv = {}
for n in data['graph']:
    kv = n.get('kv', {})
    p = kv.get('p', 'UNKNOWN')
    by_kv[p] = by_kv.get(p, 0) + 1
print(json.dumps(by_kv, indent=2))
print(f'Total: {len(data[\"graph\"])}')
" "$@"
