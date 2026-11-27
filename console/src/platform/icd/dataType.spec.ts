// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";
import { join } from "node:path";

import { icd } from "@synnaxlabs/client";
import { caseconv, payload } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { ICD } from "@/platform/icd";

// jsdom replaces URL with a class that readFileSync does not take, so the path is a
// string.
const TABLE_PATH = join(
  import.meta.dirname,
  "../../../../core/pkg/service/icd/testdata/data_types.json",
);

const objectZ = z.record(z.string(), z.unknown());
const defaultZ = z.object({ field: objectZ, dataType: z.string() });
const checkZ = defaultZ.extend({ accepted: z.boolean() });
const byFormatZ = <T extends z.ZodType>(caseZ: T) =>
  z.record(payload.formatZ, caseZ.array());

const tableZ = z.object({
  payloads: z.record(payload.formatZ, objectZ),
  fields: z.record(payload.formatZ, objectZ),
  reads: byFormatZ(checkZ),
  writes: checkZ.array(),
  defaults: byFormatZ(defaultZ),
});

const table = tableZ.parse(
  caseconv.snakeToCamel(JSON.parse(readFileSync(TABLE_PATH, "utf8"))),
);

const payloadOf = (format: payload.Format, field: Record<string, unknown>) =>
  icd.payloadZ.parse({
    ...table.payloads[format],
    fields: [{ ...table.fields[format], ...field }],
  });

const fieldOf = (format: payload.Format, field: Record<string, unknown>): ICD.Field =>
  payloadOf(format, field).fields[0];

const jsonFieldOf = (field: Record<string, unknown>): icd.JSONField => {
  const p = payloadOf("json", field);
  if (p.format !== "json") throw new Error(`payload is ${p.format}, not json`);
  return p.fields[0];
};

const casesOf = <C>(byFormat: Record<payload.Format, C[]>) =>
  payload.FORMATS.flatMap((format) => byFormat[format].map((c) => ({ format, ...c })));

describe("ICD data types", () => {
  it.each(casesOf(table.reads))(
    "should report whether a $format field $field converts to $dataType",
    ({ format, field, dataType, accepted }) => {
      expect(ICD.converts(fieldOf(format, field), dataType)).toBe(accepted);
    },
  );

  it.each(table.writes)(
    "should report whether a JSON field $field takes $dataType",
    ({ field, dataType, accepted }) => {
      expect(ICD.encodes(jsonFieldOf(field), dataType)).toBe(accepted);
    },
  );

  it.each(casesOf(table.defaults))(
    "should give a binding of a $format field $field that names no data type $dataType",
    ({ format, field, dataType }) => {
      expect(ICD.bindingDataType({ dataType: "" }, fieldOf(format, field))).toBe(
        dataType,
      );
    },
  );

  it("should keep the data type of a binding that names one", () => {
    expect(
      ICD.bindingDataType({ dataType: "float32" }, fieldOf("json", { type: "float" })),
    ).toBe("float32");
  });

  it("should give a binding with an absent data type the default of its field", () => {
    expect(ICD.bindingDataType({}, fieldOf("json", { type: "integer" }))).toBe("int64");
  });
});
