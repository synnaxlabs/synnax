// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { type json, map, type payload, uuid } from "@synnaxlabs/x";

import { fieldNames } from "@/platform/icd/fieldNames";

type Inferred = Pick<json.InferredField, "pointer" | "kind" | "timeFormat">;

/** An inferred field with the enum labels that a read can map to numbers. */
type Labeled = Inferred & Pick<json.InferredField, "enumLabels">;

const INTEGER = /^-?(0|[1-9]\d*)$/;
const NUMBER = /^-?(0|[1-9]\d*)(\.\d+)?([eE][+-]?\d+)?$/;

const spellNumbers = (labels: string[]): boolean => labels.every((l) => NUMBER.test(l));

const createField = (
  { pointer, kind, timeFormat }: Inferred,
  name: string,
): icd.JSONField => icd.jsonFieldZ.parse({ name, pointer, type: kind, timeFormat });

/** @returns a field for each inferred field, named after its pointer. */
export const createFields = (inferred: Inferred[]): icd.JSONField[] => {
  const names = fieldNames(inferred.map(({ pointer }) => pointer));
  return inferred.map((f, i) => createField(f, names[i]));
};

/**
 * @returns the kind of the field that createReadField makes from an inferred field. A
 * field with enum labels is a number, so that its values plot.
 */
export const readKind = ({
  kind,
  enumLabels: labels,
}: Pick<Labeled, "kind" | "enumLabels">): payload.Kind => {
  if (labels.length === 0) return kind;
  if (spellNumbers(labels) && !labels.every((l) => INTEGER.test(l))) return "float";
  return "integer";
};

/**
 * @returns a field named name that reads the values of an inferred field. Enum labels
 * that all spell numbers read as those numbers. Other labels read as their positions.
 */
export const createReadField = (inferred: Labeled, name: string): icd.JSONField => {
  const { pointer, enumLabels: labels } = inferred;
  const type = readKind(inferred);
  if (labels.length === 0 || spellNumbers(labels))
    return createField({ ...inferred, kind: type }, name);
  const enumValues = labels.map((label, i) => ({ label, value: i.toString() }));
  return icd.jsonFieldZ.parse({ name, pointer, type, enumValues });
};

/**
 * @returns one inferred field for each pointer of fields, in the order of the first
 * field at that pointer. It is that first field with the sorted labels of every field
 * at the pointer, or with no labels when one of them has none.
 */
export const mergeFields = <
  F extends Pick<json.InferredField, "pointer" | "enumLabels">,
>(
  fields: F[],
): F[] => {
  const byPointer = new Map<string, F[]>();
  for (const f of fields) map.getOrInsert(byPointer, f.pointer, () => []).push(f);
  return [...byPointer.values()].map((same) => {
    const labels = same.map(({ enumLabels: l }) => l);
    if (labels.some((l) => l.length === 0)) return { ...same[0], enumLabels: [] };
    return { ...same[0], enumLabels: [...new Set(labels.flat())].sort() };
  });
};

/** @returns a read field for each inferred field, named after its pointer. */
export const createReadFields = (inferred: Labeled[]): icd.JSONField[] => {
  const names = fieldNames(inferred.map(({ pointer }) => pointer));
  return inferred.map((f, i) => createReadField(f, names[i]));
};

/** @returns the name of the kind of document that holds the field at present. */
export const kindOf = (present: string): string => fieldNames([present])[0];

/**
 * @returns the name of the message for one kind of the documents of a source named
 * name: the name, then the name of present, the field that only that kind holds.
 */
export const kindName = (name: string, present?: string): string =>
  present == null ? name : `${name} ${kindOf(present)}`;

/**
 * @returns name, or name with a number when a message in taken holds it, such as
 * `plant/oven 2`.
 */
export const availableName = (name: string, taken: icd.Message[]): string => {
  const names = new Set(taken.map((m) => m.name));
  let next = name;
  for (let n = 2; names.has(next); n++) next = `${name} ${n}`;
  return next;
};

const baseName = (name: string): string => (name === "" ? "Untitled" : name);

/**
 * @returns name for a new message, or `Untitled` when name is empty. A name that a
 * message in taken holds gets a number, as availableName gives.
 */
export const messageName = (name: string, taken: icd.Message[]): string =>
  availableName(baseName(name), taken);

/** @returns whether messageName can give name for base, with or without a number. */
export const namedBy = (name: string, base: string): boolean => {
  const given = baseName(base);
  if (name === given) return true;
  return name.startsWith(`${given} `) && /^\d+$/.test(name.slice(given.length + 1));
};

/** @returns a JSON message of fields, named as messageName gives. */
export const createMessage = (
  name: string,
  fields: icd.JSONField[],
  taken: icd.Message[],
): icd.Message => ({
  key: uuid.create(),
  name: messageName(name, taken),
  payload: { format: "json", fields },
});
