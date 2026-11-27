// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { payload, TimeStamp } from "@synnaxlabs/x";

type Type = icd.JSONField["type"];

type Default<T extends Type> = NonNullable<
  Extract<icd.JSONField, { type: T }>["default"]
>;

const BASE64 = /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/;

const PARSERS: { [T in Type]: (text: string) => Default<T> } = {
  float: (text) => {
    const value = Number(text);
    if (text.trim() === "" || !Number.isFinite(value))
      throw new Error("Enter a number.");
    return value;
  },
  integer: (text) => {
    const result = payload.integerZ.safeParse(text.trim());
    if (!result.success) throw new Error("Enter a whole number.");
    return result.data;
  },
  unsigned: (text) => {
    const result = payload.unsignedZ.safeParse(text.trim());
    if (!result.success) throw new Error("Enter a whole number of 0 or more.");
    return result.data;
  },
  string: (text) => text,
  bytes: (text) => {
    if (!BASE64.test(text)) throw new Error("Enter base64.");
    return text;
  },
  boolean: (text) => {
    const word = text.trim().toLowerCase();
    if (word !== "true" && word !== "false") throw new Error("Enter true or false.");
    return word === "true";
  },
  timestamp: (text) => {
    if (text.trim().toLowerCase() === "now") return { type: "now" };
    try {
      return { type: "constant", value: new TimeStamp(text.trim()) };
    } catch {
      throw new Error("Enter now or a date.");
    }
  },
  uuid: (text) => {
    const word = text.trim();
    if (word.toLowerCase() === "random") return { type: "random" };
    const result = icd.uuidDefaultZ.safeParse({ type: "constant", value: word });
    if (!result.success) throw new Error("Enter random or a UUID.");
    return result.data;
  },
  json: (text) => {
    try {
      JSON.parse(text);
    } catch {
      throw new Error("Enter JSON.");
    }
    return text;
  },
};

/**
 * @returns field with the default that text spells, or field as it is when text is
 * empty. A field with enum values also takes the label of a value.
 * @throws an Error that says what to enter when text spells no default of the field.
 */
export const parseDefault = (field: icd.JSONField, text: string): icd.JSONField => {
  if (text === "") return field;
  const mapped =
    "enumValues" in field
      ? field.enumValues.find(({ label }) => label === text)?.value
      : undefined;
  const value: unknown = PARSERS[field.type](mapped == null ? text : String(mapped));
  return icd.jsonFieldZ.parse({ ...field, default: value });
};
