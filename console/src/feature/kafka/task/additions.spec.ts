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

import { createAdditions } from "@/feature/kafka/task/additions";

const createNumber = (pointer: string, dataType: string): json.InferredField =>
  json.inferredFieldZ.parse({
    pointer,
    dataType,
    type: "number",
    enum: [],
    count: 1,
    mismatched: 0,
  });

describe("createAdditions", () => {
  it("should give each number field the type of its inferred data type", () => {
    const fields = [
      createNumber("/count", "int64"),
      createNumber("/id", "uint64"),
      createNumber("/level", "float64"),
    ];
    const shape = json.shapeZ.parse({ sources: ["pump"], fields, index: -1, count: 1 });
    const picks = new Map([[0, new Set(fields.map(({ pointer }) => pointer))]]);
    const [{ message }] = createAdditions([shape], picks, [], [], (key) => key);
    expect(message.payload).toMatchObject({
      fields: [
        { pointer: "/count", type: "integer" },
        { pointer: "/id", type: "unsigned" },
        { pointer: "/level", type: "float" },
      ],
    });
  });
});
