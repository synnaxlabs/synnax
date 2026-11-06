// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const OPERATORS: ReadonlyArray<[string, string]> = [
  ["eq", "="],
  ["in", "in"],
  ["contains", "contains"],
  ["startsWith", "starts with"],
  ["regex", "matches"],
  ["gt", ">"],
  ["gte", ">="],
  ["lt", "<"],
  ["lte", "<="],
  ["any", "has any of"],
];

const OPERATOR_KEYS = new Set([...OPERATORS.map(([op]) => op), "caseInsensitive"]);

const NODE_KEYS = new Set(["and", "or", "not"]);

type Node = Record<string, unknown>;

/** One term of the terms that a node joins with `and`. */
interface Conjunct {
  text: string;
  disjunction: boolean;
}

const unwrap = (value: unknown): unknown =>
  value != null &&
  typeof value === "object" &&
  "toJSON" in value &&
  typeof value.toJSON === "function"
    ? unwrap(value.toJSON())
    : value;

const json = (value: unknown): string => {
  const v = unwrap(value);
  if (Array.isArray(v)) return `[${v.map(json).join(",")}]`;
  if (v != null && typeof v === "object")
    return `{${Object.entries(v)
      .filter(([, e]) => e !== undefined)
      .sort(([a], [b]) => (a < b ? -1 : 1))
      .map(([k, e]) => `${JSON.stringify(k)}:${json(e)}`)
      .join(",")}}`;
  if (typeof v === "string") return JSON.stringify(v);
  return String(v);
};

const literal = (value: unknown): string => {
  const v = unwrap(value);
  return Array.isArray(v) ? `[${v.map(literal).join(", ")}]` : json(v);
};

const isOperators = (value: unknown): value is Node =>
  value != null &&
  typeof value === "object" &&
  Object.getPrototypeOf(value) === Object.prototype &&
  Object.keys(value).every((k) => OPERATOR_KEYS.has(k));

const clauses = (field: string, value: unknown): string[] => {
  if (!isOperators(value)) return [`${field} = ${literal(value)}`];
  const suffix = value.caseInsensitive === true ? " (ignoring case)" : "";
  return OPERATORS.filter(([op]) => value[op] != null).map(([op, word]) => {
    const operand = op === "regex" ? `/${String(value[op])}/` : literal(value[op]);
    return `${field} ${word} ${operand}${suffix}`;
  });
};

const join = (terms: Conjunct[]): string =>
  terms
    .map(({ text, disjunction }) =>
      disjunction && terms.length > 1 ? `(${text})` : text,
    )
    .join(" and ");

const disjunct = (terms: Conjunct[]): string => {
  if (terms.length === 0) return "true";
  if (terms.length === 1) return terms[0].text;
  return `(${join(terms)})`;
};

const conjuncts = (node: Node): Conjunct[] => {
  const terms: Conjunct[] = Object.keys(node)
    .filter((k) => !NODE_KEYS.has(k) && node[k] != null)
    .sort()
    .flatMap((field) => clauses(field, node[field]))
    .map((text) => ({ text, disjunction: false }));
  const { and, or, not } = node as { and?: Node[]; or?: Node[]; not?: Node };
  if (and != null) terms.push(...and.flatMap(conjuncts));
  if (or != null)
    if (or.length === 0) terms.push({ text: "false", disjunction: false });
    else if (or.length === 1) terms.push(...conjuncts(or[0]));
    else
      terms.push({
        text: or.map((child) => disjunct(conjuncts(child))).join(" or "),
        disjunction: true,
      });
  if (not != null) {
    const inner = conjuncts(not);
    const text = inner.length === 0 ? "true" : join(inner);
    terms.push({ text: `not (${text})`, disjunction: false });
  }
  return terms;
};

/**
 * @returns a short, readable description of the records that tree selects, such as
 * `name starts with "press" and active = true`. An empty tree returns an empty string.
 */
export const describe = (tree: object): string => join(conjuncts(tree as Node));
