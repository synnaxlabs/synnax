// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import * as location from "@/spatial/location/location";

describe("Location", () => {
  describe("construction", () => {
    [
      ["from valueOf", String("left")],
      ["from string", "left"],
      ["from direction", "x"],
    ].forEach(([name, arg]) =>
      test(name, () => expect(location.construct(arg)).toEqual("left")),
    );
  });
});
