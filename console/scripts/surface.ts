// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

type SliceSchemas = Record<string, z.ZodType>;

export interface Scopes {
  global: SliceSchemas;
  core: SliceSchemas;
  project: SliceSchemas;
  window: SliceSchemas;
  transient: string[];
}

const flatten = (value: unknown, path: string): string[] => {
  if (Array.isArray(value) && value.every((v) => typeof v !== "object"))
    return [`${path} ${JSON.stringify(value.toSorted())}`];
  if (typeof value !== "object" || value == null || Object.keys(value).length === 0)
    return [`${path} ${JSON.stringify(value)}`];
  return Object.entries(value).flatMap(([k, v]) => flatten(v, `${path}.${k}`));
};

const describeSlice = (path: string, schema: z.ZodType): string[] => {
  const { $schema, ...json } = z.toJSONSchema(schema, {
    io: "input",
    unrepresentable: "any",
    cycles: "ref",
    reused: "inline",
  });
  return flatten(json, path);
};

/**
 * @returns the shape of each stored slice as sorted lines, one per field of its JSON
 * schema, so the reports of two revisions differ only when saved state does.
 */
export const report = (scopes: Scopes): string => {
  const { transient, ...stored } = scopes;
  const lines = [
    ...transient.map((slice) => `transient ${slice}`),
    ...Object.entries(stored).flatMap(([scope, slices]) =>
      Object.entries(slices).flatMap(([slice, schema]) =>
        describeSlice(`${scope}.${slice}`, schema),
      ),
    ),
  ];
  return `${lines.toSorted().join("\n")}\n`;
};
