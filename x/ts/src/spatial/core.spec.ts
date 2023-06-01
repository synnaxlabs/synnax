// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { LooseXYT, XY } from "@/spatial";

describe("XY", () => {
  describe("construction", () => {
    [
      ["from object", { x: 1, y: 2 }],
      ["from couple", [1, 2]],
      ["from dimensions", { width: 1, height: 2 }],
      ["from signed dimensions", { signedWidth: 1, signedHeight: 2 }],
    ].forEach(([name, arg]) => {
      test(name as string, () => {
        const xy = new XY(arg as LooseXYT);
        expect(xy.x).toEqual(1);
        expect(xy.y).toEqual(2);
      });
    });
  });
});
