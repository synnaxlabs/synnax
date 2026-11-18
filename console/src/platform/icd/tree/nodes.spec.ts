// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

const read = (message: string) => ({ message, bindings: [] });

describe("ICD.Tree", () => {
  describe("initialExpanded", () => {
    it("should expand every source and each message of a source that reads several", () => {
      expect(
        ICD.Tree.initialExpanded([
          { key: "a", messages: [read("m1")] },
          { key: "b", messages: [read("m2"), read("m3")] },
          { key: "tag" },
        ]),
      ).toEqual(["a", "b", "b/m2", "b/m3", "tag"]);
    });
  });
});
