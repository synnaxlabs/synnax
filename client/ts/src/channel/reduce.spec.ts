// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { actionZ } from "@/channel/actions";
import { createOf } from "@/channel/reduce";

describe("channel reduce", () => {
  describe("createOf", () => {
    it("should read a created channel that carries no expression", () => {
      const action = actionZ.parse({
        type: "create",
        create: {
          channel: {
            name: "doubled",
            leaseholder: 1,
            dataType: "float32",
            isIndex: false,
            localKey: 42,
            localIndex: 0,
            virtual: true,
            concurrency: 0,
          },
        },
      });
      expect(createOf(action)).toMatchObject({
        key: ((1 << 20) | 42) >>> 0,
        name: "doubled",
        expression: "",
        operations: [],
      });
    });
  });
});
