// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { id } from "@/id";
import { status } from "@/status";
import { TimeStamp } from "@/telem";

describe("status", () => {
  describe("create", () => {
    it("should create a status", () => {
      const s = status.create({ variant: "success", message: "test" });
      expect(s.key).toHaveLength(id.LENGTH);
      expect(s.time).toBeInstanceOf(TimeStamp);
      expect(s.time.beforeEq(TimeStamp.now())).toBeTruthy();
    });
  });
});
