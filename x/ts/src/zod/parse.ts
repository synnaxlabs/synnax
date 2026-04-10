// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { deep } from "@/deep";
import { errors } from "@/errors";
import { fmt } from "@/fmt";
import { primitive } from "@/primitive";
import { type status } from "@/status";

const DEFAULT_LABEL = "value";
const PARSE_ERROR_TYPE = "zod.parse";
const MARKER = "✗";
const MAX_ALIGN_WIDTH = 60;

export interface ParseOptions {
  label?: string;
  context?: Record<string, unknown>;
}

// Tight formatting options for sibling context values rendered inline in a parent
// view. The faithful version of the input lives on `err.input` and in
// `toStatus().details.input` for callers that need the full structure.
const PARENT_VIEW_OPTIONS: fmt.Options = {
  maxStringLength: 60,
  maxArrayLength: 3,
  maxDepth: 3,
};

const parentPath = (path: ReadonlyArray<PropertyKey>): ReadonlyArray<PropertyKey> =>
  path.slice(0, -1);

const issueOrigin = (issue: z.core.$ZodIssue): string =>
  "origin" in issue ? String(issue.origin) : "value";

/**
 * Returns a concise "expected X" phrase for a single zod issue. This is the core
 * description used everywhere: in the margin-annotated parent view as the reason
 * text, and as a prefix for the flat root-level form that appends `, received Y`.
 *
 * The `received` value is intentionally NOT part of this phrase; callers that need
 * to show it append it separately depending on whether they have a local value or
 * the root itself is the failing one.
 */
const describeCore = (issue: z.core.$ZodIssue): string => {
  switch (issue.code) {
    case "invalid_type": {
      const expected = "expected" in issue ? String(issue.expected) : "value";
      return `expected ${expected}`;
    }
    case "invalid_value": {
      const values = "values" in issue ? JSON.stringify(issue.values) : "<allowed>";
      return `expected one of ${values}`;
    }
    case "unrecognized_keys":
      return "unexpected key";
    case "too_small":
    case "too_big": {
      const inclusive = "inclusive" in issue ? issue.inclusive !== false : true;
      const side = issue.code === "too_small" ? "small" : "large";
      const op =
        issue.code === "too_small" ? (inclusive ? ">=" : ">") : inclusive ? "<=" : "<";
      const limit =
        issue.code === "too_small"
          ? "minimum" in issue
            ? issue.minimum
            : "?"
          : "maximum" in issue
            ? issue.maximum
            : "?";
      return `${issueOrigin(issue)} too ${side}: expected ${op}${limit}`;
    }
    case "invalid_format": {
      const format = "format" in issue ? String(issue.format) : "format";
      return `expected ${format} format`;
    }
    case "not_multiple_of": {
      const divisor = "divisor" in issue ? String(issue.divisor) : "?";
      return `expected multiple of ${divisor}`;
    }
    case "custom":
      return issue.message || "custom validation failed";
    default:
      return issue.message;
  }
};

/**
 * A "expected X, received Y" phrase for root-level issues where the whole input is
 * the failing value. Appends the received value from `root` to the core description.
 */
const describeFlat = (
  issue: z.core.$ZodIssue,
  root: unknown,
  options: fmt.Options,
): string => {
  // `unrecognized_keys` at the root can only happen before expansion; after
  // `expandUnrecognizedKeys` runs in `format()`, such issues have path.length > 0
  // and take the parent-view path. Handle them here only for completeness: if they
  // somehow reach the flat form, render the raw keys list.
  if (
    issue.code === "unrecognized_keys" &&
    "keys" in issue &&
    Array.isArray(issue.keys)
  )
    return `unknown keys: ${issue.keys.join(", ")}`;
  const received = JSON.stringify(fmt.value(root, options));
  return `${describeCore(issue)}, received ${received}`;
};

/**
 * Returns the maximum path depth reached by any issue within a list, recursing into
 * nested invalid_union / invalid_key / invalid_element branches. Used to pick the
 * union branch that got furthest before failing, which is usually the variant the
 * caller actually intended.
 */
const branchDepth = (issues: ReadonlyArray<z.core.$ZodIssue>): number => {
  let max = 0;
  for (const issue of issues) {
    if (issue.path.length > max) max = issue.path.length;
    if (issue.code === "invalid_union" && "errors" in issue)
      for (const branch of issue.errors) {
        const d = branchDepth(branch);
        if (d > max) max = d;
      }
    if (
      (issue.code === "invalid_key" || issue.code === "invalid_element") &&
      "issues" in issue
    ) {
      const d = branchDepth(issue.issues);
      if (d > max) max = d;
    }
  }
  return max;
};

/**
 * Walks a list of zod issues and expands every nested-issue container into its
 * leaves, with the outer path prepended:
 *
 * - `invalid_union` → picks the deepest-reaching branch and flattens it
 * - `invalid_element` → flattens nested issues, prepending the failing element's
 *   key as a path segment (so errors point at `record[badKey]` directly)
 * - `invalid_key` → flattens nested issues at the outer path, since the "bad key"
 *   has no distinct position in the input
 *
 * The result is a flat list of leaf issues pointing at specific locations in the
 * input.
 */
const flattenIssues = (
  issues: ReadonlyArray<z.core.$ZodIssue>,
  basePath: ReadonlyArray<PropertyKey> = [],
): z.core.$ZodIssue[] => {
  const out: z.core.$ZodIssue[] = [];
  for (const issue of issues) {
    const fullPath = [...basePath, ...issue.path];
    if (
      issue.code === "invalid_union" &&
      "errors" in issue &&
      issue.errors.length > 0
    ) {
      let best = issue.errors[0];
      let bestDepth = branchDepth(best);
      for (let i = 1; i < issue.errors.length; i++) {
        const d = branchDepth(issue.errors[i]);
        if (d > bestDepth) {
          best = issue.errors[i];
          bestDepth = d;
        }
      }
      out.push(...flattenIssues(best, fullPath));
      continue;
    }
    if (
      issue.code === "invalid_element" &&
      "issues" in issue &&
      "key" in issue &&
      (typeof issue.key === "string" || typeof issue.key === "number")
    ) {
      out.push(...flattenIssues(issue.issues, [...fullPath, issue.key]));
      continue;
    }
    if (issue.code === "invalid_key" && "issues" in issue) {
      out.push(...flattenIssues(issue.issues, fullPath));
      continue;
    }
    out.push({ ...issue, path: fullPath } as z.core.$ZodIssue);
  }
  return out;
};

/**
 * Expands unrecognized_keys issues into per-key issues so each bad key becomes its
 * own mark in the parent view. After expansion each synthesized issue points at a
 * single offending key via its path.
 */
const expandUnrecognizedKeys = (
  issues: ReadonlyArray<z.core.$ZodIssue>,
): z.core.$ZodIssue[] => {
  const out: z.core.$ZodIssue[] = [];
  for (const issue of issues) {
    if (
      issue.code === "unrecognized_keys" &&
      "keys" in issue &&
      Array.isArray(issue.keys)
    ) {
      for (const key of issue.keys)
        out.push({ ...issue, path: [...issue.path, key] } as z.core.$ZodIssue);
      continue;
    }
    out.push(issue);
  }
  return out;
};

interface Mark {
  reason: string;
  missing: boolean;
}

/**
 * Renders an object or array with margin markers (`✗`) on the keys that have issues,
 * annotating each bad key with the expected type. Synthetic `<missing>` entries are
 * appended for keys the schema expected but the input didn't contain.
 *
 * `baseIndent` is the column at which the opening brace sits. Marked lines place the
 * marker two columns to the left of where the key's quote begins so that content
 * still aligns with unmarked lines.
 *
 * For array parents, the truncation window is extended to include any marked
 * index so that the reader can always see the annotated element.
 */
const renderParentView = (
  parent: unknown,
  marks: Map<string, Mark>,
  baseIndent: string,
): string[] => {
  const inner = `${baseIndent}  `;
  const marked = `${baseIndent + MARKER} `;

  // For array parents, ensure the truncation window includes all marked indices
  // so the ✗ lines aren't silently dropped when a bad entry lies past the default
  // maxArrayLength.
  let options = PARENT_VIEW_OPTIONS;
  if (Array.isArray(parent)) {
    let maxIdx = -1;
    for (const key of marks.keys()) {
      const idx = Number(key);
      if (Number.isInteger(idx) && idx > maxIdx) maxIdx = idx;
    }
    const defaultMax = PARENT_VIEW_OPTIONS.maxArrayLength ?? 3;
    if (maxIdx >= defaultMax)
      options = { ...PARENT_VIEW_OPTIONS, maxArrayLength: maxIdx + 2 };
  }

  // Sanitize the whole parent via fmt.value so sibling key redaction (passwords,
  // tokens) runs against the parent's own keys rather than per-value, which would
  // have no key context to match against.
  const sanitized = fmt.value(parent, options);

  // Fallback for non-container parents. Only reachable when the union flattener
  // picks a branch whose parent path doesn't exist in the input at all: we still
  // want to show the reader *something*, so we render the failing value and the
  // first mark's reason on a single marker line.
  if (sanitized == null || typeof sanitized !== "object") {
    const val = JSON.stringify(sanitized);
    const first = marks.values().next().value;
    const reason = first ? `  × ${first.reason}` : "";
    return [`${marked}${val}${reason}`];
  }

  const lines: string[] = [];
  const isArray = Array.isArray(sanitized);
  lines.push(`${baseIndent}${isArray ? "[" : "{"}`);

  interface Entry {
    content: string;
    mark?: Mark;
  }

  const entries: Entry[] = [];

  if (isArray) {
    const arr = sanitized as unknown[];
    for (let i = 0; i < arr.length; i++) {
      const mark = marks.get(String(i));
      entries.push({
        content: JSON.stringify(arr[i]),
        mark: mark && !mark.missing ? mark : undefined,
      });
    }
  } else {
    const obj = sanitized as Record<string, unknown>;
    for (const k of Object.keys(obj)) {
      const mark = marks.get(k);
      entries.push({
        content: `"${k}": ${JSON.stringify(obj[k])}`,
        mark: mark && !mark.missing ? mark : undefined,
      });
    }
    // Synthetic missing entries for fields the schema expected but the input lacks.
    for (const [k, mark] of marks.entries())
      if (mark.missing)
        entries.push({
          content: `"${k}": <missing>`,
          mark,
        });
  }

  // Compute alignment width for the reasons, capped so that one abnormally long
  // marked line doesn't push the column way to the right.
  let alignWidth = 0;
  entries.forEach((e, i) => {
    if (e.mark == null) return;
    const trailing = i < entries.length - 1 ? 1 : 0;
    const w = e.content.length + trailing;
    if (w > alignWidth && w <= MAX_ALIGN_WIDTH) alignWidth = w;
  });

  entries.forEach((e, i) => {
    const comma = i < entries.length - 1 ? "," : "";
    const line = e.content + comma;
    if (e.mark != null) {
      const pad = " ".repeat(Math.max(2, alignWidth - line.length + 2));
      lines.push(`${marked}${line}${pad}× ${e.mark.reason}`);
    } else lines.push(`${inner}${line}`);
  });

  lines.push(baseIndent + (isArray ? "]" : "}"));
  return lines;
};

const ROOT_GROUP = "__root__";

const renderIssues = (
  issues: ReadonlyArray<z.core.$ZodIssue>,
  root: unknown,
  options: fmt.Options,
): string => {
  // Group issues by parent path in the order they first appear, preserving zod's
  // declaration order rather than alphabetizing.
  const groups = new Map<string, z.core.$ZodIssue[]>();
  const order: string[] = [];
  for (const issue of issues) {
    const key =
      issue.path.length === 0 ? ROOT_GROUP : `::${fmt.path(parentPath(issue.path))}`;
    let group = groups.get(key);
    if (group == null) {
      group = [];
      groups.set(key, group);
      order.push(key);
    }
    group.push(issue);
  }

  const blocks: string[] = [];
  for (const key of order) {
    const group = groups.get(key) as z.core.$ZodIssue[];

    // Root-level issues (path.length === 0): flat bullets, no parent view.
    if (key === ROOT_GROUP) {
      blocks.push(group.map((i) => `  × ${describeFlat(i, root, options)}`).join("\n"));
      continue;
    }

    const parentArr = parentPath(group[0].path);
    const { present, value: parent } = deep.atKeys(root, parentArr);

    // If the union flattener picked a branch whose parent doesn't exist in the
    // input, we can't render a parent view. Fall back to a flat at/× block.
    if (!present) {
      blocks.push(
        group
          .map(
            (i) => `  at ${fmt.path(i.path)}\n    × ${describeFlat(i, root, options)}`,
          )
          .join("\n"),
      );
      continue;
    }

    // Build a marks map keyed by the last path segment so renderParentView can
    // annotate each bad key in place.
    const marks = new Map<string, Mark>();
    for (const issue of group) {
      const lastKey = String(issue.path[issue.path.length - 1]);
      const found = deep.atKeys(root, issue.path);
      marks.set(lastKey, {
        reason: describeCore(issue),
        missing: !found.present,
      });
    }

    const isRootParent = parentArr.length === 0;
    const baseIndent = isRootParent ? "  " : "    ";
    const viewLines = renderParentView(parent, marks, baseIndent);
    if (isRootParent) blocks.push(viewLines.join("\n"));
    else blocks.push([`  at ${fmt.path(parentArr)}`, ...viewLines].join("\n"));
  }

  return blocks.join("\n\n");
};

const formatContextLine = (
  context: Record<string, unknown>,
  options: fmt.Options,
): string => {
  const entries = Object.entries(context);
  if (entries.every(([, v]) => primitive.is(v)))
    return entries
      .map(([k, v]) => `${k}=${typeof v === "string" ? v : String(v)}`)
      .join(", ");
  return fmt.stringify(context, options);
};

interface FormatArgs {
  issues: ReadonlyArray<z.core.$ZodIssue>;
  input: unknown;
  label: string;
  context?: Record<string, unknown>;
  formatOptions?: fmt.Options;
}

const format = ({
  issues,
  input,
  label,
  context,
  formatOptions,
}: FormatArgs): string => {
  const opts = formatOptions ?? {};
  const flat = expandUnrecognizedKeys(flattenIssues(issues));
  const count = flat.length === 1 ? "1 issue" : `${flat.length} issues`;
  const parts: string[] = [`Failed to parse ${label} (${count})`];
  parts.push(renderIssues(flat, input, opts));
  if (context != null && Object.keys(context).length > 0)
    parts.push(`  context: ${formatContextLine(context, opts)}`);
  return parts.join("\n\n");
};

export interface ParseErrorArgs {
  issues: ReadonlyArray<z.core.$ZodIssue>;
  input: unknown;
  label: string;
  context?: Record<string, unknown>;
  cause?: unknown;
  formatOptions?: fmt.Options;
}

/**
 * An error thrown by `zod.parse` when a value fails to parse. It retains the original
 * input, a human-readable label, and optional context so that callers and the status
 * system can render a richer failure message than a raw `ZodError`.
 *
 * Extends the typed error system in `@/errors` so callers can match against it with
 * `ParseError.matches(err)` rather than `instanceof`, which is robust across worker
 * boundaries and network hops.
 *
 * Note that `err.issues.length` and the `(N issues)` count shown in `err.message` can
 * differ: the message counts "leaves" after union/element flattening and
 * unrecognized-keys expansion, while `err.issues` exposes the original zod array
 * unchanged for programmatic consumers.
 */
export class ParseError
  extends errors.createTyped(PARSE_ERROR_TYPE)
  implements status.Custom
{
  readonly issues: ReadonlyArray<z.core.$ZodIssue>;
  readonly input: unknown;
  readonly label: string;
  readonly context?: Record<string, unknown>;

  constructor({ issues, input, label, context, cause, formatOptions }: ParseErrorArgs) {
    super(format({ issues, input, label, context, formatOptions }), { cause });
    this.issues = issues;
    this.input = input;
    this.label = label;
    this.context = context;
  }

  toStatus(): Partial<status.Crude<z.ZodRecord, "error">> {
    const details: Record<string, unknown> = {
      input: fmt.value(this.input),
      issues: this.issues,
    };
    if (this.context != null) details.context = this.context;
    return {
      message: `Failed to parse ${this.label}`,
      description: this.message,
      details,
    };
  }
}

interface ParseErrorPayload {
  label: string;
  context?: Record<string, unknown>;
  issues: ReadonlyArray<unknown>;
  input: unknown;
}

errors.register({
  encode: (err) => {
    if (!ParseError.matches(err) || !(err instanceof ParseError)) return null;
    // Defensive: zod 4 rarely populates issue.input on standard issues, but
    // custom refinements or third-party check plugins may set it. Redact it
    // through fmt.value on the way out so no transport leaks a secret even if
    // one does slip in.
    const payload: ParseErrorPayload = {
      label: err.label,
      context: err.context,
      issues: err.issues.map((i) => ({ ...i, input: fmt.value(i.input) })),
      input: fmt.value(err.input),
    };
    return { type: PARSE_ERROR_TYPE, data: JSON.stringify(payload) };
  },
  decode: (payload) => {
    if (payload.type !== PARSE_ERROR_TYPE) return null;
    const parsed = JSON.parse(payload.data) as ParseErrorPayload;
    return new ParseError({
      issues: parsed.issues as ReadonlyArray<z.core.$ZodIssue>,
      input: parsed.input,
      label: parsed.label,
      context: parsed.context,
    });
  },
});

/**
 * Parses `value` against `schema`. On failure, throws a `ParseError` that retains the
 * original input along with a human-readable label and optional context fields. The
 * error's `message` is a pre-formatted breakdown suitable for logs and status display.
 */
export const parse = <S extends z.ZodType>(
  schema: S,
  value: unknown,
  options: ParseOptions = {},
): z.infer<S> => {
  const result = schema.safeParse(value);
  if (result.success) return result.data;
  throw new ParseError({
    issues: result.error.issues,
    input: value,
    label: options.label ?? DEFAULT_LABEL,
    context: options.context,
    cause: result.error,
  });
};
