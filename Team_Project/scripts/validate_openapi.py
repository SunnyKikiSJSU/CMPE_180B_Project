#!/usr/bin/env python3
"""Lightweight OpenAPI structure check.

This is a development helper, not a runtime dependency. It verifies that
openapi.yaml parses, that every local $ref resolves, and that operationIds are
unique. For full specification validation, run a dedicated OpenAPI linter.

Usage:
    python scripts/validate_openapi.py docs/api/openapi.yaml
"""

import sys
from collections import Counter


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: validate_openapi.py <openapi.yaml>", file=sys.stderr)
        return 2

    try:
        import yaml
    except ImportError:
        print("PyYAML is not installed; install it with: pip install pyyaml")
        return 2

    with open(sys.argv[1], encoding="utf-8") as handle:
        document = yaml.safe_load(handle)

    if not isinstance(document, dict):
        print("document root is not a mapping")
        return 1

    references = []
    operation_ids = []

    def walk(node):
        if isinstance(node, dict):
            for key, value in node.items():
                if key == "$ref" and isinstance(value, str):
                    references.append(value)
                if key == "operationId" and isinstance(value, str):
                    operation_ids.append(value)
                walk(value)
        elif isinstance(node, list):
            for item in node:
                walk(item)

    walk(document)

    def resolve(reference):
        if not reference.startswith("#/"):
            return True
        node = document
        for raw in reference[2:].split("/"):
            part = raw.replace("~1", "/").replace("~0", "~")
            if not isinstance(node, dict) or part not in node:
                return False
            node = node[part]
        return True

    missing = sorted({ref for ref in references if not resolve(ref)})
    duplicates = sorted(
        name for name, count in Counter(operation_ids).items() if count > 1
    )
    paths = document.get("paths", {})
    methods = {"get", "post", "put", "patch", "delete", "head", "options"}

    print(f"openapi: {document.get('openapi')}")
    print(f"paths: {len(paths)}")
    print(f"operations: {len(operation_ids)}")
    print(f"refs: {len(references)}")
    print(f"missing refs: {missing}")
    print(f"duplicate operationIds: {duplicates}")

    return 1 if missing or duplicates else 0


if __name__ == "__main__":
    raise SystemExit(main())
