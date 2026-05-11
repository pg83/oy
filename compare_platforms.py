#!/usr/bin/env python3
"""
Platform distribution comparison script for T-140 investigation.
Compares current generated graph vs reference graph platform assignments.
"""

import json
import sys
from collections import defaultdict, Counter

def load_graph(path):
    with open(path, 'r') as f:
        data = json.load(f)
    return data.get('graph', [])

def analyze_platforms(graph):
    """Analyze platform distribution by node and module characteristics."""
    by_platform = Counter()
    by_module = defaultdict(set)
    by_type = defaultdict(Counter)
    by_tag = defaultdict(set)

    for node in graph:
        platform = node.get('platform', 'unknown')
        node_type = node.get('type', 'unknown')
        tags = node.get('tags', [])
        props = node.get('target_properties', {})
        module_path = props.get('path', props.get('module_dir', 'unknown'))

        # Count by platform
        by_platform[platform] += 1

        # Track platforms per module
        by_module[module_path].add(platform)

        # Count node types by platform
        by_type[node_type][platform] += 1

        # Track platforms for tagged nodes
        for tag in tags:
            by_tag[tag].add(platform)

    return by_platform, by_module, by_type, by_tag

def print_platform_distribution(title, by_platform):
    print(f"\n{title}")
    print("=" * 60)
    total = sum(by_platform.values())
    for platform, count in sorted(by_platform.items()):
        pct = (count / total * 100) if total > 0 else 0
        print(f"  {platform:30s}: {count:4d} ({pct:5.1f}%)")
    print(f"  {'TOTAL':30s}: {total:4d}")

def print_node_type_distribution(title, by_type):
    print(f"\n{title}")
    print("=" * 70)
    for node_type, platforms in sorted(by_type.items()):
        total = sum(platforms.values())
        line = f"  {node_type:10s}: "
        for platform, count in sorted(platforms.items()):
            pct = (count / total * 100) if total > 0 else 0
            line += f"{platform}={count}({pct:.0f}%) "
        print(line)

def print_tag_distribution(title, by_tag):
    print(f"\n{title}")
    print("=" * 60)
    for tag, platforms in sorted(by_tag.items()):
        print(f"  {tag}: {sorted(platforms)}")

def print_module_platform_summary(by_module):
    print(f"\nModule Platform Summary")
    print("=" * 60)

    single_aarch64 = []
    single_x86_64 = []
    dual_platform = []

    for module, platforms in by_module.items():
        if len(platforms) == 1:
            if 'aarch64' in str(platforms):
                single_aarch64.append(module)
            elif 'x86_64' in str(platforms):
                single_x86_64.append(module)
        elif len(platforms) == 2:
            dual_platform.append(module)

    print(f"  Single-platform aarch64 modules: {len(single_aarch64)}")
    for m in sorted(single_aarch64)[:10]:
        print(f"    {m}")

    print(f"\n  Single-platform x86_64 modules: {len(single_x86_64)}")
    for m in sorted(single_x86_64)[:10]:
        print(f"    {m}")

    print(f"\n  Dual-platform modules: {len(dual_platform)}")
    for m in sorted(dual_platform)[:10]:
        print(f"    {m}")

def compare_platforms(ref_graph, curr_graph):
    """Compare platform distributions between reference and current."""
    print("\n" + "=" * 70)
    print("PLATFORM DISTRIBUTION COMPARISON")
    print("=" * 70)

    ref_by_platform, ref_by_module, ref_by_type, ref_by_tag = analyze_platforms(ref_graph)
    curr_by_platform, curr_by_module, curr_by_type, curr_by_tag = analyze_platforms(curr_graph)

    print_platform_distribution("REFERENCE Graph:", ref_by_platform)
    print_platform_distribution("CURRENT Graph:", curr_by_platform)

    print_platform_distribution("Platform Difference:", Counter(ref_by_platform) - Counter(curr_by_platform))

    print_node_type_distribution("Node Type Distribution (Reference):", ref_by_type)
    print_node_type_distribution("Node Type Distribution (Current):", curr_by_type)

    print_tag_distribution("Tag Platforms (Reference):", ref_by_tag)
    print_tag_distribution("Tag Platforms (Current):", curr_by_tag)

    print_module_platform_summary(ref_by_module)
    print_module_platform_summary(curr_by_module)

def main():
    if len(sys.argv) != 3:
        print("Usage: python3 compare_platforms.py <reference.json> <current.json>")
        sys.exit(1)

    ref_path = sys.argv[1]
    curr_path = sys.argv[2]

    print(f"Loading reference graph: {ref_path}")
    ref_graph = load_graph(ref_path)

    print(f"Loading current graph: {curr_path}")
    curr_graph = load_graph(curr_path)

    compare_platforms(ref_graph, curr_graph)

if __name__ == '__main__':
    main()
