// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import * as direction from "@/spatial/direction/direction";

describe("Direction", () => {
  describe("construction", () => {
    type T = [string, direction.Crude];
    const TESTS: T[] = [
      ["from location", "top"],
      ["from literal", "y"],
    ];
    TESTS.forEach(([name, arg]) =>
      test(name, () => expect(direction.construct(arg)).toEqual("y")),
    );
  });
});
