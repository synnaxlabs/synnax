// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { type json, uuid } from "@synnaxlabs/x";

import { fieldNames } from "@/platform/icd/fieldNames";

type Inferred = Pick<json.InferredField, "pointer" | "kind" | "timeFormat">;

/** @returns a field named name that reads the values of an inferred field. */
export const createField = (
  { pointer, kind, timeFormat }: Inferred,
  name: string,
): icd.JSONField => icd.jsonFieldZ.parse({ name, pointer, type: kind, timeFormat });

/** @returns a field for each inferred field, named after its pointer. */
export const createFields = (inferred: Inferred[]): icd.JSONField[] => {
  const names = fieldNames(inferred.map(({ pointer }) => pointer));
  return inferred.map((f, i) => createField(f, names[i]));
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
