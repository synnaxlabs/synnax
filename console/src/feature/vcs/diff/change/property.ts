// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";

/** How one property or element differs between two versions. */
export type Kind = "added" | "removed" | "changed";

/** One piece of a value as text, such as one channel of a list of channels. */
export interface Part {
  text: string;
  /** resource is the resource the text names. Absent for plain text. */
  resource?: resource.ID;
}

/** One difference between two versions of a resource. */
export interface Property {
  /** path names the property, outermost first, such as ["Channels", "ai_0", "Port"]. */
  path: string[];
  kind: Kind;
  /** before is the old value. Absent for an added element. */
  before?: Part[];
  /** after is the new value. Absent for a removed element. */
  after?: Part[];
}

/** Returns the name of the resource, or undefined if it does not exist. */
export type Name = (id: resource.ID) => string | undefined;

export interface DiffOptions {
  name?: Name;
  /** omit names top-level fields to leave out, such as fields shown elsewhere. */
  omit?: string[];
}

const ACRONYMS: Record<string, string> = {
  ai: "AI",
  ao: "AO",
  api: "API",
  di: "DI",
  hz: "Hz",
  id: "ID",
  iepe: "IEPE",
  ip: "IP",
  json: "JSON",
  ni: "NI",
  opc: "OPC",
  rtd: "RTD",
  tcp: "TCP",
  ua: "UA",
  udp: "UDP",
  url: "URL",
  uuid: "UUID",
};

/** Labels for fields whose words alone do not read well, keyed in snake_case. */
const FIELD_LABELS: Record<string, string> = {
  auto_start: "Auto start",
  current_excit_source: "Excitation source",
  current_excit_val: "Excitation (A)",
  gage_factor: "Gage factor",
  max_val: "Maximum",
  min_val: "Minimum",
  nominal_gage_resistance: "Nominal resistance (Ω)",
  sample_rate: "Sample rate (Hz)",
  strain_config: "Bridge configuration",
  stream_rate: "Stream rate (Hz)",
  terminal_config: "Terminal configuration",
  voltage_excit_source: "Excitation source",
  voltage_excit_val: "Excitation (V)",
};

/**
 * Turns a field name in snake_case or camelCase into sentence case words.
 * @example humanize("sample_rate") === "Sample rate (Hz)"
 * @example humanize("stateChannel") === "State channel"
 */
export const humanize = (field: string): string => {
  const label =
    FIELD_LABELS[field.replace(/([a-z0-9])([A-Z])/g, "$1_$2").toLowerCase()];
  if (label != null) return label;
  const words = field
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .split(/[_\s-]+/)
    .filter((w) => w.length > 0)
    .map((w) => w.toLowerCase());
  return words
    .map((w, i) => {
      const acronym = ACRONYMS[w];
      if (acronym != null) return acronym;
      return i === 0 ? w.charAt(0).toUpperCase() + w.slice(1) : w;
    })
    .join(" ");
};

const CHANNEL_FIELD = /^(channel|channels|index)$|(_c|C)hannels?$/;

/** Returns the resource that a value of the field names, or null if it names none. */
const referenceOf = (field: string, value: unknown): resource.ID | null => {
  if (typeof value === "number" && value > 0) {
    if (CHANNEL_FIELD.test(field)) return { type: "channel", key: String(value) };
    if (field === "rack") return { type: "rack", key: String(value) };
  }
  if (typeof value === "string" && value.length > 0 && field === "device")
    return { type: "device", key: value };
  return null;
};

const isPlainObject = (v: unknown): v is Record<string, unknown> =>
  v != null && typeof v === "object" && !Array.isArray(v);

// A task stores its config as JSON text, so the text is read as the object it holds.
const expand = (value: unknown): unknown => {
  if (typeof value !== "string") return value;
  const trimmed = value.trimStart();
  if (!trimmed.startsWith("{") && !trimmed.startsWith("[")) return value;
  try {
    const parsed: unknown = JSON.parse(value);
    return typeof parsed === "object" && parsed != null ? parsed : value;
  } catch {
    return value;
  }
};

const elementKey = (v: unknown): string | null => {
  if (!isPlainObject(v)) return null;
  const { key } = v;
  if (typeof key === "string" || typeof key === "number") return String(key);
  return null;
};

/** Collects every resource the values name, such as channels and devices, once each. */
export const references = (...values: unknown[]): resource.ID[] => {
  const ids = new Map<string, resource.ID>();
  const walk = (value: unknown, field: string | null): void => {
    const v = expand(value);
    if (Array.isArray(v)) {
      v.forEach((el) => walk(el, field));
      return;
    }
    if (isPlainObject(v)) {
      Object.entries(v).forEach(([k, el]) => walk(el, k));
      return;
    }
    const id = field == null ? null : referenceOf(field, v);
    if (id != null) ids.set(resource.idToString(id), id);
  };
  values.forEach((v) => walk(v, null));
  return [...ids.values()];
};

// Arrays of objects become records keyed by a token per element, so elements match
// by key rather than position. Tokens never contain the "." or "[" deep.difference
// uses to join a path.
const TOKEN_PREFIX = "\u0001";

interface Normalized {
  value: unknown;
  labels: Map<string, string>;
}

const elementLabel = (el: unknown, fallback: string, name?: Name): string => {
  if (!isPlainObject(el)) return fallback;
  const { name: elementName, channel, port } = el;
  if (typeof elementName === "string" && elementName.length > 0) return elementName;
  if (typeof channel === "number") {
    const resolved = name?.({ type: "channel", key: String(channel) });
    if (resolved != null) return resolved;
  }
  if (typeof port === "number" || typeof port === "string") return `Port ${port}`;
  return fallback;
};

const normalize = (
  before: unknown,
  after: unknown,
  name?: Name,
): [Normalized, Normalized] => {
  const labels = new Map<string, string>();
  const tokens = new Map<string, string>();
  const tokenFor = (id: string, label: string): string => {
    let token = tokens.get(id);
    if (token == null) {
      token = `${TOKEN_PREFIX}${tokens.size}`;
      tokens.set(id, token);
      labels.set(token, label);
    }
    return token;
  };
  const walk = (value: unknown, path: string): unknown => {
    const v = expand(value);
    if (Array.isArray(v)) {
      if (!v.some(isPlainObject)) return v;
      const keys = v.map(elementKey);
      const keyed = keys.every((k) => k != null) && new Set(keys).size === keys.length;
      const out: Record<string, unknown> = {};
      v.forEach((el, i) => {
        const id = keyed ? `${path}#${keys[i]}` : `${path}[${i}]`;
        const fallback = keyed ? String(keys[i]) : `Item ${i + 1}`;
        const token = tokenFor(id, elementLabel(el, fallback, name));
        out[token] = walk(el, id);
      });
      return out;
    }
    if (isPlainObject(v)) {
      const out: Record<string, unknown> = {};
      Object.entries(v).forEach(([k, el]) => {
        out[k] = walk(el, `${path}.${k}`);
      });
      return out;
    }
    return v;
  };
  return [
    { value: walk(before, ""), labels },
    { value: walk(after, ""), labels },
  ];
};

const splitPath = (path: string): string[] =>
  path
    .split(".")
    .flatMap((seg) => {
      const parts: string[] = [];
      const head = seg.replace(/\[(\d+)\]/g, (_, i: string) => {
        parts.push(`[${i}]`);
        return "";
      });
      return [head, ...parts];
    })
    .filter((s) => s.length > 0);

const MAX_LIST = 6;

const plain = (text: string): Part[] => [{ text }];

const formatValue = (value: unknown, field: string, name?: Name): Part[] => {
  if (value == null) return plain("None");
  if (typeof value === "boolean") return plain(value ? "Yes" : "No");
  if (Array.isArray(value)) {
    if (value.length === 0) return plain("None");
    const parts = value
      .slice(0, MAX_LIST)
      .flatMap((v, i) => [
        ...(i > 0 ? plain(", ") : []),
        ...formatValue(v, field, name),
      ]);
    const rest = value.length - MAX_LIST;
    return rest > 0 ? [...parts, ...plain(` and ${rest} more`)] : parts;
  }
  const id = referenceOf(field, value);
  if (id != null) {
    // A resource that no longer exists shows its key, with nothing to open.
    const resolved = name?.(id);
    return resolved == null ? plain(id.key) : [{ text: resolved, resource: id }];
  }
  if (typeof value === "number") return plain(String(value));
  if (typeof value === "string") return plain(value.length === 0 ? "Empty" : value);
  return plain(JSON.stringify(value));
};

/** Joins the parts of a value into its text. */
export const text = (parts: Part[]): string => parts.map((p) => p.text).join("");

// The top-level config of a task wraps every field, so it adds no meaning to a path.
const TRANSPARENT = new Set(["config"]);

const humanizeSegment = (seg: string, labels: Map<string, string>): string => {
  const label = labels.get(seg);
  if (label != null) return label;
  const index = /^\[(\d+)\]$/.exec(seg);
  if (index != null) return `Item ${Number(index[1]) + 1}`;
  return humanize(seg);
};

// A nested field that repeats its parent's name, such as label.label, reads as one.
const humanizePath = (segments: string[], labels: Map<string, string>): string[] =>
  segments
    .map((s) => humanizeSegment(s, labels))
    .filter((s, i, all) => i === 0 || s !== all[i - 1]);

const strip = (value: unknown, omit: string[]): Record<string, unknown> => {
  const expanded = expand(value);
  if (!isPlainObject(expanded)) return {};
  const out = { ...expanded };
  omit.forEach((k) => delete out[k]);
  return out;
};

const describePath = (
  raw: string[],
  labels: Map<string, string>,
): { path: string[]; field: string } => {
  const kept = raw.filter((s, i) => i > 0 || !TRANSPARENT.has(s) || raw.length === 1);
  const field = raw.findLast((s) => !labels.has(s) && !s.startsWith("[")) ?? "";
  return { path: humanizePath(kept, labels), field };
};

/**
 * Lists every property that differs between two versions of a resource, with
 * human-readable paths and values. Elements of an array of objects match by their key
 * when every element has one, and by position otherwise. JSON text, such as a task
 * config, is compared as the object it holds. The key of a channel, device, or rack
 * reads as its name when it exists.
 */
export const diffProperties = (
  before: unknown,
  after: unknown,
  { name, omit = [] }: DiffOptions = {},
): Property[] => {
  const [b, a] = normalize(strip(before, omit), strip(after, omit), name);
  const labels = new Map([...b.labels, ...a.labels]);
  const diffs = deep.difference(b.value as object, a.value as object);
  const properties: Property[] = [];
  Object.entries(diffs).forEach(([key, [prev, next]]) => {
    if (deep.equal(prev, next)) return;
    const { path, field } = describePath(splitPath(key), labels);
    if (prev == null && isPlainObject(next)) properties.push({ path, kind: "added" });
    else if (next == null && isPlainObject(prev))
      properties.push({ path, kind: "removed" });
    else
      properties.push({
        path,
        kind: "changed",
        before: formatValue(prev, field, name),
        after: formatValue(next, field, name),
      });
  });
  return properties;
};

/**
 * Lists every property of a resource that one version has and the other lacks, such as
 * a resource that was created or deleted. Paths and values read as in
 * {@link diffProperties}.
 * @param kind - "added" puts each value in after, "removed" in before.
 */
export const listProperties = (
  value: unknown,
  kind: "added" | "removed",
  { name, omit = [] }: DiffOptions = {},
): Property[] => {
  const stripped = strip(value, omit);
  const [{ value: normalized, labels }] = normalize(stripped, stripped, name);
  const properties: Property[] = [];
  const walk = (v: unknown, raw: string[]): void => {
    if (isPlainObject(v) && Object.keys(v).length > 0) {
      Object.entries(v).forEach(([k, el]) => walk(el, [...raw, k]));
      return;
    }
    // A key names its element, which the path already shows.
    if (raw.length === 0 || raw.at(-1) === "key") return;
    const { path, field } = describePath(raw, labels);
    const parts = isPlainObject(v) ? plain("None") : formatValue(v, field, name);
    properties.push(
      kind === "added" ? { path, kind, after: parts } : { path, kind, before: parts },
    );
  };
  walk(normalized, []);
  return properties;
};

const PATH_SEPARATOR = " › ";

/** Joins the path of a property for display. */
export const formatPath = ({ path }: Pick<Property, "path">): string =>
  path.join(PATH_SEPARATOR);

/**
 * Formats a property as one line of text.
 * @example format({path: ["Sample rate"], kind: "changed", before: "1000", after:
 * "2500"}) === "Sample rate: 1000 -> 2500"
 */
export const format = (property: Property): string => {
  const path = formatPath(property);
  const { before, after } = property;
  switch (property.kind) {
    case "added":
      return after == null ? `${path}: added` : `${path}: ${text(after)}`;
    case "removed":
      return before == null ? `${path}: removed` : `${path}: ${text(before)}`;
    case "changed":
      if (before == null || after == null) return `${path}: changed`;
      return `${path}: ${text(before)} -> ${text(after)}`;
  }
};
