// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { json, type payload } from "@synnaxlabs/x";

/** A field of the documents of a shape, as a sample shows it. */
export interface Field extends Pick<
  json.InferredField,
  "pointer" | "kind" | "type" | "timeFormat" | "optional"
> {
  /** The JSON text of the value that every document holds, when the sample tells. */
  value?: string;
}

/** The documents of a source that hold the same fields, as a sample shows them. */
export interface Shape {
  fields: Field[];
  /** The pointer of the field that only this shape holds. Absent for an only shape. */
  present?: string;
}

/** @returns the value at pointer in doc, or undefined when doc holds none. */
const valueAt = (doc: unknown, pointer: string): unknown => {
  let value = doc;
  for (const token of json.parsePointer(pointer)) {
    if (value == null || typeof value !== "object") return undefined;
    value = (value as Record<string, unknown>)[token];
  }
  return value;
};

/**
 * @returns shape as reuse reads it. A field gets the value that each of documents holds
 * when they all hold one.
 * @param documents - The sampled documents of the shape.
 */
export const shapeOf = (shape: json.Shape, documents: unknown[]): Shape => ({
  fields: shape.fields.map(({ pointer, kind, type, timeFormat, optional }) => {
    const values = new Set(documents.map((d) => JSON.stringify(valueAt(d, pointer))));
    const [value] = values;
    const one = values.size === 1 && value != null;
    return { pointer, kind, type, timeFormat, optional, ...(one ? { value } : {}) };
  }),
  present: shape.present == null ? undefined : shape.fields[shape.present].pointer,
});

/** A message that fits a shape. */
export interface Fit {
  message: icd.Message;
  /** The pointers of the fields of the shape that the message lacks. */
  missing: string[];
}

const NUMBERS: readonly payload.Kind[] = ["float", "integer", "unsigned"];

/** @returns whether field holds the values at pointer. */
export const covers = (field: icd.JSONField, pointer: string): boolean =>
  field.pointer === pointer ||
  (field.type === "json" && pointer.startsWith(`${field.pointer}/`));

/** @returns whether field reads the values of sampled. */
export const reads = (field: icd.JSONField, sampled: Field): boolean => {
  switch (field.type) {
    case "json":
      return true;
    case "string":
      return sampled.type === "string";
    case "float":
      return NUMBERS.includes(sampled.kind);
    case "integer":
      return sampled.kind === "integer" || sampled.kind === "unsigned";
    case "timestamp":
      return sampled.kind === "timestamp" && sampled.timeFormat === field.timeFormat;
    default:
      return sampled.kind === field.type;
  }
};

/** @returns whether rule takes every document of shape. */
const takes = (shape: Shape, rule?: icd.JSONMatch): boolean => {
  if (rule == null) return shape.present == null;
  const held = shape.fields.filter(({ optional }) => optional !== true);
  if (rule.type === "present")
    return held.some(
      ({ pointer }) =>
        pointer === rule.pointer || pointer.startsWith(`${rule.pointer}/`),
    );
  const value = held.find(({ pointer }) => pointer === rule.pointer)?.value;
  return value != null && value === JSON.stringify(rule.value);
};

/**
 * @returns how message fits shape, or nothing when it does not. A JSON message fits
 * when its match rule takes every document of the shape, each of its fields with no
 * default is in every document, each of its fields reads the values it holds, and it
 * holds a field of the shape. It may lack fields of the shape.
 */
const fit = (shape: Shape, message: icd.Message): Fit | undefined => {
  const { payload } = message;
  if (payload.format !== "json" || !takes(shape, payload.match)) return undefined;
  for (const field of payload.fields) {
    const held = shape.fields.filter(({ pointer }) => covers(field, pointer));
    const everywhere = held.some(({ optional }) => optional !== true);
    if (!everywhere && field.default == null) return undefined;
    if (!held.every((sampled) => reads(field, sampled))) return undefined;
  }
  const missing = shape.fields
    .filter(({ pointer }) => !payload.fields.some((f) => covers(f, pointer)))
    .map(({ pointer }) => pointer);
  if (missing.length === shape.fields.length) return undefined;
  return { message, missing };
};

/** A source of a task, such as a topic, as a sample shows it. */
export interface Source {
  shapes: Shape[];
  /** The keys of the messages that the device stores channels for at the source. */
  stored: ReadonlySet<icd.MessageKey>;
}

/** The messages of one ICD that fit the shapes of a source best. */
export interface Candidate {
  icd: icd.Key;
  /** The name of the ICD. */
  name: string;
  /**
   * The messages that fit each shape of the source best, by position. Two or more
   * messages tie.
   */
  fits: Fit[][];
}

/** The ICD that a source reads, and the fit of each of its shapes by position. */
export interface Reading {
  icd: icd.Key;
  /** The name of the ICD. */
  name: string;
  /** Absent for a shape that no message fits, or that two messages fit equally. */
  fits: (Fit | undefined)[];
}

/** The ICD messages that a source reads. */
export interface Result {
  /** Absent when no ICD fits, or when the best one ties. */
  reading?: Reading;
  /** The ICDs whose messages fit the source best, home first. Two or more tie. */
  candidates: Candidate[];
}

/** How well messages fit: stored messages first, then fields held. */
interface Score {
  stored: number;
  held: number;
}

const compare = (a: Score, b: Score): number => b.stored - a.stored || b.held - a.held;

/** @returns the candidate of the messages of an ICD for source, and its score. */
const rank = (
  { key, name, messages }: Pick<icd.ICD, "key" | "name" | "messages">,
  { shapes, stored }: Source,
): { candidate: Candidate; score: Score } => {
  const score: Score = { stored: 0, held: 0 };
  const fits = shapes.map((shape) => {
    const scored = messages
      .flatMap((m) => fit(shape, m) ?? [])
      .map((f) => ({
        f,
        score: {
          stored: stored.has(f.message.key) ? 1 : 0,
          held: shape.fields.length - f.missing.length,
        },
      }))
      .sort((a, b) => compare(a.score, b.score));
    const [best] = scored;
    if (best == null) return [];
    score.stored += best.score.stored;
    score.held += best.score.held;
    return scored.filter((s) => compare(s.score, best.score) === 0).map(({ f }) => f);
  });
  return { candidate: { icd: key, name, fits }, score };
};

/**
 * @returns the ICD messages that source reads. A source reads one ICD: the one whose
 * messages the device stores the most channels for, then that holds the most fields of
 * the source, then home, the ICD that most sources of the task read. It gives no
 * reading when two ICDs still tie.
 */
export const find = (
  source: Source,
  icds: Pick<icd.ICD, "key" | "name" | "messages">[],
  home?: icd.Key,
): Result => {
  const isHome = ({ candidate }: { candidate: Candidate }) =>
    candidate.icd === home ? 1 : 0;
  const ranked = icds
    .map((ic) => rank(ic, source))
    .filter(({ candidate }) => candidate.fits.some((f) => f.length > 0))
    .sort((a, b) => compare(a.score, b.score) || isHome(b) - isHome(a));
  const [first, second] = ranked;
  if (first == null) return { candidates: [] };
  const candidates = ranked
    .filter(({ score }) => compare(score, first.score) === 0)
    .map(({ candidate }) => candidate);
  const tied =
    second != null && compare(first.score, second.score) === 0 && isHome(first) === 0;
  if (tied) return { candidates };
  const { icd: key, name, fits } = first.candidate;
  const reading = {
    icd: key,
    name,
    fits: fits.map((f) => (f.length === 1 ? f[0] : undefined)),
  };
  return { reading, candidates };
};
