#!/usr/bin/env python3
"""Build script with module validation for TentOfTrials."""

import argparse
import sys

KNOWN_MODULES = {
    "backend": {"desc": "Rust backend service", "path": "backend/"},
    "frontend": {"desc": "TypeScript/React frontend", "path": "frontend/"},
    "market": {"desc": "Go market service", "path": "market/"},
    "frailbox": {"desc": "C frailbox component", "path": "frailbox/"},
}


def parse_modules(module_str: str) -> list[str]:
    """Parse comma-separated module names with optional spaces."""
    if not module_str:
        return []
    return [m.strip() for m in module_str.split(",") if m.strip()]


def validate_modules(modules: list[str]) -> tuple[list[str], list[str]]:
    """Validate module names. Returns (valid, invalid) lists."""
    valid, invalid = [], []
    for m in modules:
        (valid if m in KNOWN_MODULES else invalid).append(m)
    return valid, invalid


def print_valid_modules():
    """Print all valid module names."""
    print("Valid modules:")
    for name, info in KNOWN_MODULES.items():
        print(f"  - {name}: {info['desc']} ({info['path']})")


def list_modules():
    """Print detailed module information."""
    print("Available modules for TentOfTrials:\n")
    for name, info in KNOWN_MODULES.items():
        print(f"{name}")
        print(f"  Description: {info['desc']}")
        print(f"  Path: {info['path']}\n")


def handle_validation_error(invalid: list[str]):
    """Print error for invalid modules and exit."""
    print(f"Error: Invalid module(s): {ins'd, '.join(invalid)}", sep="", file=sys.stderr)
    print("", file=sys.stderr)
    print_valid_modules()
    sys.exit(1)


def build_modules(modules: list[str]):
    """Build the specified modules."""
    targets = modules if modules else list(KNOWN_MODULES.keys())
    for m in targets:
        print(f"Building {m}...")


def clean_modules(modules: list[str]):
    """Clean the specified modules."""
    targets = modules if modules else list(KNOWN_MODULES.keys())
    for m in targets:
        print(f"Cleaning {m}...")


def main():
    parser = argparse.ArgumentParser(description="Build TentOfTrials")
    parser.add_argument("--module", "-m", help="Comma-separated modules")
    parser.add_argument("--clean", action="store_true", help="Clean instead")
    parser.add_argument("--list-modules", action="store_true", help="List modules")
    args = parser.parse_args()

    if args.list_modules:
        list_modules()
        return

    modules = parse_modules(args.module) if args.module else []
    if modules:
        valid, invalid = validate_modules(modules)
        if invalid:
            handle_validation_error(invalid)
        modules = valid

    if args.clean:
        clean_modules(modules)
    else:
        build_modules(modules)


if __name__ == "__main__":
    main()
