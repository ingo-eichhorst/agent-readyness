"""Utility functions for the application."""

import os
import json
from typing import Dict, List, Optional


class DataProcessor:
    """Processes data records with various transformations."""

    def __init__(self, config: Dict[str, str]):
        self.config = config
        self.cache: Dict[str, str] = {}

    def process_record(self, record: Dict, strict: bool = False) -> Optional[Dict]:
        """Process a single record with validation and transformation.

        This function has high cyclomatic complexity due to multiple
        branches and conditions.
        """
        if not record:
            return None

        result = {}
        status = record.get("status", "unknown")

        if status == "active":
            result["active"] = True
            if "priority" in record:
                if record["priority"] > 5:
                    result["urgent"] = True
                elif record["priority"] > 3:
                    result["normal"] = True
                else:
                    result["low"] = True
        elif status == "inactive":
            result["active"] = False
            if strict:
                return None
        elif status == "pending":
            result["pending"] = True
            for key in record:
                if key.startswith("meta_"):
                    result[key] = record[key]
        else:
            result["unknown"] = True

        if "tags" in record:
            for tag in record["tags"]:
                if tag == "important" or tag == "critical":
                    result["flagged"] = True
                    break

        while "next" in record:
            record = record["next"]
            if record.get("status") == "done":
                break

        return result

    def validate(self, data: List[Dict]) -> bool:
        """Validate a list of records."""
        return all(isinstance(r, dict) for r in data)


def simple_add(a: int, b: int) -> int:
    """Add two numbers together."""
    return a + b


def load_config(path: str) -> Dict:
    """Load configuration from a JSON file."""
    with open(path, "r") as f:
        return json.load(f)


def format_output(items: List[str], separator: str = ", ") -> str:
    """Format a list of items into a string."""
    return separator.join(items)


def find_duplicates(items: List[str]) -> List[str]:
    """Find duplicate items in a list."""
    seen = set()
    duplicates = []
    for item in items:
        if item in seen:
            duplicates.append(item)
        else:
            seen.add(item)
    return duplicates
