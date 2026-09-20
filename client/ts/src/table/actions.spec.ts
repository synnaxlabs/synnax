// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { scopedActionZ } from "@/table/actions.gen";

describe("table action wire decode", () => {
  test("preserves cell keys through the set-channel schema", () => {
    const wire = {
      key: "c211c9f3-0b3c-4f5a-9185-e9cabc09a091",
      dispatch_key: "abc",
      seq: 1,
      actions: [
        {
          type: "create",
          create: {
            table: {
              key: "c211c9f3-0b3c-4f5a-9185-e9cabc09a091",
              name: "Metrics Table",
              rows: [{ size: 36, cells: ["UnSv19BHjPB", "SMm7XhRkCJM"] }],
              columns: [{ size: 72 }, { size: 72 }],
              cells: {
                UnSv19BHjPB: { key: "UnSv19BHjPB", variant: "value", props: {} },
                SMm7XhRkCJM: { key: "SMm7XhRkCJM", variant: "value", props: {} },
              },
            },
          },
        },
      ],
    };
    const raw = new TextEncoder().encode(JSON.stringify(wire));
    const buf = new ArrayBuffer(4 + raw.byteLength);
    new DataView(buf).setUint32(0, raw.byteLength, true);
    new Uint8Array(buf).set(raw, 4);
    const s = new Series({ data: buf, dataType: DataType.JSON });
    const [parsed] = s.parseJSON(scopedActionZ);
    const action = parsed.actions[0];
    expect(action.type).toEqual("create");
    if (action.type !== "create") return;
    expect(Object.keys(action.create.table.cells)).toEqual([
      "UnSv19BHjPB",
      "SMm7XhRkCJM",
    ]);
    expect(action.create.table.cells.UnSv19BHjPB.key).toEqual("UnSv19BHjPB");
  });
});
