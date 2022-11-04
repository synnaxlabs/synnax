from typing import Any


def determine_suggestions(data: list[dict[str, Any]], fields: list[str]) -> list[str]:
    if len(data) == 0:
        return []

    suggestions = []
    for field in fields:
        for value in [row[field] for row in data]:
            if value not in suggestions:
                suggestions.append(value)
    return suggestions
