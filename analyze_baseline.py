#!/usr/bin/env python3
"""
Graph analyzer for T-186 baseline validation.

Analyzes reference and generated graphs to produce node type and platform distributions.
Uses kv.p field for node type detection (correct method).
"""

import json
import sys
from collections import defaultdict, Counter
from pathlib import Path

def load_graph(path):
    """Load graph JSON file."""
    with open(path, 'r') as f:
        data = json.load(f)

    # Handle both list and dict with 'graph' field formats
    if isinstance(data, dict) and 'graph' in data:
        return data['graph']
    elif isinstance(data, list):
        return data
    else:
        raise ValueError(f"Unknown graph format in {path}")

def extract_node_type(node):
    """Extract node type from kv.p field."""
    if 'kv' in node and 'p' in node['kv']:
        return node['kv']['p']
    return 'UNKNOWN'

def extract_platform(node):
    """Extract platform from platform field or target_properties.platform."""
    if 'platform' in node:
        return node['platform']
    if 'target_properties' in node and 'platform' in node['target_properties']:
        return node['target_properties']['platform']
    return 'UNKNOWN'

def analyze_graph(graph):
    """Analyze graph and return node type and platform distributions."""
    node_types = Counter()
    platforms = Counter()
    type_by_platform = defaultdict(Counter)

    for node in graph:
        node_type = extract_node_type(node)
        platform = extract_platform(node)

        node_types[node_type] += 1
        platforms[platform] += 1
        type_by_platform[platform][node_type] += 1

    return node_types, platforms, type_by_platform

def compare_graphs(ref_graph, gen_graph):
    """Compare reference and generated graphs."""
    ref_types, ref_platforms, ref_type_by_platform = analyze_graph(ref_graph)
    gen_types, gen_platforms, gen_type_by_platform = analyze_graph(gen_graph)

    print(f"{'=' * 70}")
    print(f"Graph Comparison Results")
    print(f"{'=' * 70}")
    print(f"")
    print(f"Total Nodes:")
    print(f"  Reference: {len(ref_graph)}")
    print(f"  Generated: {len(gen_graph)}")
    print(f"  Delta: {len(gen_graph) - len(ref_graph):+d}")
    print(f"")

    print(f"Node Type Distribution:")
    all_types = sorted(set(ref_types.keys()) | set(gen_types.keys()))
    for node_type in all_types:
        ref_count = ref_types.get(node_type, 0)
        gen_count = gen_types.get(node_type, 0)
        delta = gen_count - ref_count
        status = "✓" if delta == 0 else "✗"
        print(f"  {status} {node_type:3s}: ref={ref_count:4d} gen={gen_count:4d} delta={delta:+4d}")

    print(f"")
    print(f"Platform Distribution:")
    all_platforms = sorted(set(ref_platforms.keys()) | set(gen_platforms.keys()))
    for platform in all_platforms:
        ref_count = ref_platforms.get(platform, 0)
        gen_count = gen_platforms.get(platform, 0)
        delta = gen_count - ref_count
        status = "✓" if delta == 0 else "✗"
        print(f"  {status} {platform}: ref={ref_count:4d} gen={gen_count:4d} delta={delta:+4d}")

    print(f"")
    print(f"Node Type by Platform:")
    all_platforms = sorted(set(ref_platforms.keys()) | set(gen_platforms.keys()))
    all_types = sorted(set(ref_types.keys()) | set(gen_types.keys()))
    for platform in all_platforms:
        print(f"  {platform}:")
        for node_type in all_types:
            ref_count = ref_type_by_platform[platform].get(node_type, 0)
            gen_count = gen_type_by_platform[platform].get(node_type, 0)
            if ref_count > 0 or gen_count > 0:
                delta = gen_count - ref_count
                status = "✓" if delta == 0 else "✗"
                print(f"    {status} {node_type:3s}: ref={ref_count:4d} gen={gen_count:4d} delta={delta:+4d}")
    print(f"")

    return {
        'total_ref': len(ref_graph),
        'total_gen': len(gen_graph),
        'delta': len(gen_graph) - len(ref_graph),
        'ref_types': dict(ref_types),
        'gen_types': dict(gen_types),
        'ref_platforms': dict(ref_platforms),
        'gen_platforms': dict(gen_platforms),
        'ref_type_by_platform': {k: dict(v) for k, v in ref_type_by_platform.items()},
        'gen_type_by_platform': {k: dict(v) for k, v in gen_type_by_platform.items()},
    }

def main():
    """Main entry point."""
    if len(sys.argv) < 2:
        print("Usage: analyze_baseline.py <generated_graph.json> [reference_graph.json]")
        sys.exit(1)

    gen_graph_path = sys.argv[1]
    ref_graph_path = sys.argv[2] if len(sys.argv) > 2 else "/home/pg/monorepo/yatool_orig/sg.json"

    print(f"Loading generated graph from: {gen_graph_path}")
    gen_graph = load_graph(gen_graph_path)

    print(f"Loading reference graph from: {ref_graph_path}")
    ref_graph = load_graph(ref_graph_path)

    results = compare_graphs(ref_graph, gen_graph)

    # Return non-zero if any mismatches
    if results['delta'] != 0:
        sys.exit(1)

    sys.exit(0)

if __name__ == '__main__':
    main()
