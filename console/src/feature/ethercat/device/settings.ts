// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ethercat } from "@synnaxlabs/client";
import { z } from "zod";

const ADDRESS = { index: z.number(), subIndex: z.number() };

export interface Address {
  index: number;
  subIndex: number;
}

/** Holds when the value of a setting is one of values, or above above. */
export const conditionZ = z.object({
  ...ADDRESS,
  values: z.array(z.number()).optional(),
  above: z.number().optional(),
});
export interface Condition extends z.infer<typeof conditionZ> {}

const optionZ = z.object({ value: z.number(), name: z.string() });
export interface Option extends z.infer<typeof optionZ> {}

/** A CoE object of a slave that a user sets by name. */
export const fieldZ = z.object({
  ...ADDRESS,
  name: z.string(),
  dataType: z.string(),
  default: z.number(),
  unit: z.string().default(""),
  options: z.array(optionZ).optional(),
  min: z.number().optional(),
  max: z.number().optional(),
  visibleWhen: z.array(conditionZ).default(() => []),
});
export interface Field extends z.infer<typeof fieldZ> {}

/** Rejects values that meet every condition in when but not then. */
export const requirementZ = z.object({
  when: z.array(conditionZ),
  then: conditionZ,
  message: z.string(),
});
export interface Requirement extends z.infer<typeof requirementZ> {}

/** An entry that reads an input. onlyReadBy names the one program that reads it. */
const entryZ = z.object({ ...ADDRESS, onlyReadBy: z.string().optional() });

/** The fields of one input, and the entries that read it in every PDO layout. */
export const channelZ = z.object({
  entries: z.array(entryZ),
  settings: z.array(fieldZ),
  requires: z.array(requirementZ),
});
export interface Channel extends z.infer<typeof channelZ> {}

/** The fields of a slave: those of the whole slave, and those of each input. */
export const descriptionZ = z.object({
  device: z.array(fieldZ),
  channels: z.array(channelZ),
});
export interface Description extends z.infer<typeof descriptionZ> {}

/** A value at an address, as a task or a startup parameter holds it. */
export interface Value extends Address {
  value: number;
}

/** A problem with the value of the field at the address. */
export interface Problem extends Address {
  message: string;
}

/** @returns whether the two name the same CoE object. */
export const sameAddress = (a: Address, b: Address): boolean =>
  a.index === b.index && a.subIndex === b.subIndex;

/** @returns the input that the entry at the address reads, or undefined. */
export const findChannel = (
  description: Description | undefined,
  address: Address,
): Channel | undefined =>
  description?.channels.find((c) => c.entries.some((e) => sameAddress(e, address)));

/**
 * @returns the value at the address: the one in values, or the default of the first
 * field there.
 * @throws {Error} if no field and no value is at the address.
 */
export const valueAt = (fields: Field[], values: Value[], address: Address): number => {
  const value = values.find((v) => sameAddress(v, address))?.value;
  if (value != null) return value;
  const field = fields.find((f) => sameAddress(f, address));
  if (field == null)
    throw new Error(`No setting is at ${ethercat.formatAddress(address)}`);
  return field.default;
};

const holds = (fields: Field[], values: Value[], condition: Condition): boolean => {
  const value = valueAt(fields, values, condition);
  if (condition.values != null) return condition.values.includes(value);
  return condition.above != null && value > condition.above;
};

const holdAll = (fields: Field[], values: Value[], conditions: Condition[]): boolean =>
  conditions.every((c) => holds(fields, values, c));

/** @returns the fields to show for the values. */
export const visibleFields = (fields: Field[], values: Value[]): Field[] =>
  fields.filter((f) => holdAll(fields, values, f.visibleWhen));

/**
 * @returns a setting for each address of the fields, in the order of the fields, with
 * the value in values or the default. A shown field with options takes its default
 * when its value is not one of them.
 */
export const fill = (fields: Field[], values: Value[]): ethercat.Setting[] => {
  const shown = visibleFields(fields, values);
  const settings: ethercat.Setting[] = [];
  for (const field of fields) {
    if (settings.some((s) => sameAddress(s, field))) continue;
    let value = valueAt(fields, values, field);
    const options = shown.find((f) => sameAddress(f, field))?.options;
    if (options != null && !options.some((o) => o.value === value))
      value = options.some((o) => o.value === field.default)
        ? field.default
        : options[0].value;
    const { index, subIndex, dataType } = field;
    settings.push({ index, subIndex, dataType, value });
  }
  return settings;
};

/**
 * @returns the values after the field at the address takes the value. The values of the
 * fields are filled as {@link fill} fills them, and the other values stay.
 */
export const set = (
  fields: Field[],
  values: ethercat.Setting[],
  address: Address,
  value: number,
): ethercat.Setting[] => [
  ...values.filter((v) => !fields.some((f) => sameAddress(f, v))),
  ...fill(fields, [
    { ...address, value },
    ...values.filter((v) => !sameAddress(v, address)),
  ]),
];

const rangeMessage = ({ name, unit, min, max }: Field): string => {
  const suffix = unit === "" ? "" : ` ${unit}`;
  if (min != null && max != null)
    return `${name} must be from ${min} to ${max}${suffix}.`;
  if (min != null) return `${name} must be at least ${min}${suffix}.`;
  return `${name} must be at most ${max}${suffix}.`;
};

/** @returns the problems with the values of the shown fields. */
export const check = (channel: Channel, values: Value[]): Problem[] => {
  const { settings: fields, requires } = channel;
  const problems: Problem[] = [];
  for (const field of visibleFields(fields, values)) {
    const value = valueAt(fields, values, field);
    if (
      (field.min != null && value < field.min) ||
      (field.max != null && value > field.max)
    )
      problems.push({
        index: field.index,
        subIndex: field.subIndex,
        message: rangeMessage(field),
      });
  }
  for (const { when, then, message } of requires)
    if (holdAll(fields, values, when) && !holds(fields, values, then))
      problems.push({ index: then.index, subIndex: then.subIndex, message });
  return problems;
};

/** @returns the problem with the field at the address, or undefined. */
export const problemAt = (problems: Problem[], address: Address): Problem | undefined =>
  problems.find((p) => sameAddress(p, address));
