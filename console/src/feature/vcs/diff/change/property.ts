// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";

/** How one property or element differs between two versions. */
export type Kind = "added" | "removed" | "changed";

/** One difference between two versions of a resource. */
export interface Property {
  /** path names the property, outermost first, such as ["Channels", "ai_0", "Port"]. */
  path: string[];
  kind: Kind;
  /** before is the old value as text. Absent for an added element. */
  before?: string;
  /** after is the new value as text. Absent for a removed element. */
  after?: string;
}

/** Returns the name of the channel with the given key, or undefined if unknown. */
export type ChannelName = (key: number) => string | undefined;

export interface DiffOptions {
  channelName?: ChannelName;
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

/** Returns true if the field holds the key of a channel. */
const isChannelField = (field: string): boolean => CHANNEL_FIELD.test(field);

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

/** Collects the key of every channel the values reference, in first-seen order. */
export const channelKeys = (...values: unknown[]): number[] => {
  const keys = new Set<number>();
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
    if (field != null && isChannelField(field) && typeof v === "number" && v > 0)
      keys.add(v);
  };
  values.forEach((v) => walk(v, null));
  return [...keys];
};

// Arrays of objects become records keyed by a token per element, so elements match
// by key rather than position. Tokens never contain the "." or "[" deep.difference
// uses to join a path.
const TOKEN_PREFIX = "\u0001";

interface Normalized {
  value: unknown;
  labels: Map<string, string>;
}

const elementLabel = (
  el: unknown,
  fallback: string,
  channelName?: ChannelName,
): string => {
  if (!isPlainObject(el)) return fallback;
  const { name, channel, port } = el;
  if (typeof name === "string" && name.length > 0) return name;
  if (typeof channel === "number") {
    const resolved = channelName?.(channel);
    if (resolved != null) return resolved;
  }
  if (typeof port === "number" || typeof port === "string") return `Port ${port}`;
  return fallback;
};

const normalize = (
  before: unknown,
  after: unknown,
  channelName?: ChannelName,
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
        const token = tokenFor(id, elementLabel(el, fallback, channelName));
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

const formatValue = (
  value: unknown,
  field: string,
  channelName?: ChannelName,
): string => {
  if (value == null) return "None";
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (typeof value === "number") {
    if (isChannelField(field)) return channelName?.(value) ?? String(value);
    return String(value);
  }
  if (typeof value === "string") return value.length === 0 ? "Empty" : value;
  if (Array.isArray(value)) {
    if (value.length === 0) return "None";
    const shown = value
      .slice(0, MAX_LIST)
      .map((v) => formatValue(v, field, channelName));
    const rest = value.length - shown.length;
    return rest > 0 ? `${shown.join(", ")} and ${rest} more` : shown.join(", ");
  }
  return JSON.stringify(value);
};

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
 * config, is compared as the object it holds. Channel keys read as channel names when
 * the channel is known.
 */
export const diffProperties = (
  before: unknown,
  after: unknown,
  { channelName, omit = [] }: DiffOptions = {},
): Property[] => {
  const [b, a] = normalize(strip(before, omit), strip(after, omit), channelName);
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
        before: formatValue(prev, field, channelName),
        after: formatValue(next, field, channelName),
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
  { channelName, omit = [] }: DiffOptions = {},
): Property[] => {
  const stripped = strip(value, omit);
  const [{ value: normalized, labels }] = normalize(stripped, stripped, channelName);
  const properties: Property[] = [];
  const walk = (v: unknown, raw: string[]): void => {
    if (isPlainObject(v) && Object.keys(v).length > 0) {
      Object.entries(v).forEach(([k, el]) => walk(el, [...raw, k]));
      return;
    }
    // A key names its element, which the path already shows.
    if (raw.length === 0 || raw.at(-1) === "key") return;
    const { path, field } = describePath(raw, labels);
    const text = isPlainObject(v) ? "None" : formatValue(v, field, channelName);
    properties.push(
      kind === "added" ? { path, kind, after: text } : { path, kind, before: text },
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
  switch (property.kind) {
    case "added":
      return property.after == null ? `${path}: added` : `${path}: ${property.after}`;
    case "removed":
      return property.before == null
        ? `${path}: removed`
        : `${path}: ${property.before}`;
    case "changed":
      if (property.before == null || property.after == null) return `${path}: changed`;
      return `${path}: ${property.before} -> ${property.after}`;
  }
};
