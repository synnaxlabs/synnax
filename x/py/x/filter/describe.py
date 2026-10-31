#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from dataclasses import dataclass
from typing import Any

from pydantic import BaseModel

_OPERATORS = (
    ("eq", "="),
    ("in", "in"),
    ("contains", "contains"),
    ("starts_with", "starts with"),
    ("regex", "matches"),
    ("gt", ">"),
    ("gte", ">="),
    ("lt", "<"),
    ("lte", "<="),
    ("any", "has any of"),
)

_OPERATOR_KEYS = {op for op, _ in _OPERATORS} | {"case_insensitive"}

_NODE_KEYS = {"and", "or", "not"}


@dataclass
class _Conjunct:
    text: str
    disjunction: bool = False


def _literal(value: Any) -> str:
    if isinstance(value, list):
        return "[" + ", ".join(_literal(v) for v in value) + "]"
    return json.dumps(value, separators=(",", ":"), ensure_ascii=False, sort_keys=True)


def _is_operators(value: Any) -> bool:
    return isinstance(value, dict) and value.keys() <= _OPERATOR_KEYS


def _clauses(field: str, value: Any) -> list[str]:
    if not _is_operators(value):
        return [f"{field} = {_literal(value)}"]
    suffix = " (ignoring case)" if value.get("case_insensitive") is True else ""
    return [
        f"{field} {word} {f'/{value[op]}/' if op == 'regex' else _literal(value[op])}"
        + suffix
        for op, word in _OPERATORS
        if value.get(op) is not None
    ]


def _join(terms: list[_Conjunct]) -> str:
    return " and ".join(
        f"({t.text})" if t.disjunction and len(terms) > 1 else t.text for t in terms
    )


def _disjunct(terms: list[_Conjunct]) -> str:
    if len(terms) == 0:
        return "true"
    if len(terms) == 1:
        return terms[0].text
    return f"({_join(terms)})"


def _conjuncts(node: dict[str, Any]) -> list[_Conjunct]:
    terms = [
        _Conjunct(text)
        for field in sorted(k for k in node if k not in _NODE_KEYS)
        for text in _clauses(field, node[field])
    ]
    for child in node.get("and") or []:
        terms.extend(_conjuncts(child))
    disjuncts = node.get("or")
    if disjuncts is not None:
        if len(disjuncts) == 0:
            terms.append(_Conjunct("false"))
        elif len(disjuncts) == 1:
            terms.extend(_conjuncts(disjuncts[0]))
        else:
            terms.append(
                _Conjunct(
                    " or ".join(_disjunct(_conjuncts(c)) for c in disjuncts), True
                )
            )
    negated = node.get("not")
    if negated is not None:
        inner = _conjuncts(negated)
        terms.append(_Conjunct(f"not ({_join(inner) if inner else 'true'})"))
    return terms


def describe(tree: BaseModel | None) -> str:
    """Returns a short, readable description of the records that tree selects, such as
    ``name starts with "press" and active = true``.

    :param tree: A filter tree, or None.
    :returns: The description, or an empty string for None or an empty tree.
    """
    if tree is None:
        return ""
    node = tree.model_dump(mode="json", by_alias=True, exclude_none=True)
    return _join(_conjuncts(node))
