// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import type { Action, Stored } from "@/channel/actions";
import { createOf } from "@/channel/reduce";

const create = (stored: Partial<Stored>): Action => ({
  type: "create",
  create: {
    channel: {
      name: "created",
      leaseholder: 1,
      dataType: "float32",
      isIndex: false,
      localKey: 1,
      localIndex: 0,
      virtual: true,
      concurrency: control.Concurrency.exclusive,
      expression: "",
      ...stored,
    },
  },
});

describe("channel reduce", () => {
  describe("createOf", () => {
    it("should mark a created channel with an expression calculated", () => {
      const created = createOf(create({ expression: "return 1" }));
      expect(created?.calculated).toBe(true);
    });

    it("should not mark a created virtual channel with no expression calculated", () => {
      const created = createOf(create({}));
      expect(created?.calculated).toBe(false);
    });
  });
});
