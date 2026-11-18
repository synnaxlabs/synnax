// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { json } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  createAdditions,
  defaultPicks,
  shapeKey,
} from "@/feature/kafka/task/additions";

const createField = (
  pointer: string,
  dataType: string,
  type: "number" | "string" = "number",
): json.InferredField =>
  json.inferredFieldZ.parse({
    pointer,
    dataType,
    type,
    enum: [],
    count: 1,
    mismatched: 0,
  });

const createNumber = (pointer: string, dataType: string): json.InferredField =>
  createField(pointer, dataType);

const createShape = (sources: string[], fields: json.InferredField[], index = -1) =>
  json.shapeZ.parse({ sources, fields, index, count: 1 });

const pointersOf = (shape: json.Shape): Set<string> =>
  new Set(shape.fields.map(({ pointer }) => pointer));

describe("createAdditions", () => {
  it("should give each number field the type of its inferred data type", () => {
    const fields = [
      createNumber("/count", "int64"),
      createNumber("/id", "uint64"),
      createNumber("/level", "float64"),
    ];
    const shape = json.shapeZ.parse({ sources: ["pump"], fields, index: -1, count: 1 });
    const picks = new Map([[shapeKey(shape), new Set(fields.map((f) => f.pointer))]]);
    const [{ message }] = createAdditions([shape], picks, [], [], (key) => key);
    expect(message.payload).toMatchObject({
      fields: [
        { pointer: "/count", type: "integer" },
        { pointer: "/id", type: "unsigned" },
        { pointer: "/level", type: "float" },
      ],
    });
  });

  it("should pick the fields of each shape by its key, not its position", () => {
    const oven = createShape(["oven"], [createNumber("/temperature", "float64")]);
    const pump = createShape(["pump"], [createNumber("/pressure", "float64")]);
    const picks = new Map([[shapeKey(pump), pointersOf(pump)]]);
    const additions = createAdditions([oven, pump], picks, [], [], (key) => key);
    expect(additions.map(({ recordKeys }) => recordKeys)).toEqual([["pump"]]);
  });

  it("should keep the picks of a shape when another key sends its fields", () => {
    const fields = [createNumber("/pressure", "float64")];
    const picks = new Map([
      [shapeKey(createShape(["pump-1"], fields)), new Set(["/pressure"])],
    ]);
    const shape = createShape(["pump-1", "pump-2"], fields);
    const [addition] = createAdditions([shape], picks, [], [], (key) => key);
    expect(addition.recordKeys).toEqual(["pump-1", "pump-2"]);
  });

  it("should name fields apart when their pointers give one name", () => {
    const shape = createShape(
      ["pump"],
      [createNumber("/a/b", "float64"), createNumber("/a_b", "float64")],
    );
    const picks = new Map([[shapeKey(shape), pointersOf(shape)]]);
    const [{ message }] = createAdditions([shape], picks, [], [], (key) => key);
    expect(message.payload).toMatchObject({
      fields: [{ name: "a_b" }, { name: "a_b_2" }],
    });
  });
});

describe("defaultPicks", () => {
  it("should pick the fixed fields that can share the index", () => {
    const shape = createShape(
      ["oven"],
      [
        createField("/time", "timestamp"),
        createNumber("/temperature", "float64"),
        createField("/state", "string", "string"),
      ],
      0,
    );
    expect(defaultPicks(shape)).toEqual(new Set(["/temperature"]));
  });

  it("should pick variable-length fields only when no fixed field is free", () => {
    const shape = createShape(
      ["log"],
      [
        createField("/line", "string", "string"),
        createField("/host", "string", "string"),
      ],
    );
    expect(defaultPicks(shape)).toEqual(new Set(["/line", "/host"]));
  });
});
