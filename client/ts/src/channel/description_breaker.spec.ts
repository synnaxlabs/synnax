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

import { actionZ, set } from "@/channel/actions";
import { Channel } from "@/channel/client";
import { createOf, reduceAll } from "@/channel/reduce";
import { type Payload, payloadZ } from "@/channel/types.gen";

const STORED: Payload = {
  key: (1 << 20) | 5,
  name: "pt_01",
  leaseholder: 1,
  dataType: "float32",
  isIndex: false,
  index: 0,
  virtual: true,
  internal: false,
  expression: "",
  operations: [],
  concurrency: control.Concurrency.exclusive,
  description: "Inlet pressure",
};

describe("description breaker", () => {
  it("should apply a set of the description that the Core takes", () => {
    const { next } = reduceAll(STORED, [
      set({ path: ["description"], value: "Outlet pressure" }),
    ]);
    expect(next.description).toEqual("Outlet pressure");
  });

  it("should keep the description that a create action carries", () => {
    // The action as the client reads it off the action channel.
    const create = actionZ.parse({
      type: "create",
      create: {
        channel: {
          name: "pt_01",
          leaseholder: 1,
          dataType: "float32",
          isIndex: false,
          localKey: 5,
          localIndex: 0,
          virtual: true,
          concurrency: control.Concurrency.exclusive,
          operations: [],
          expression: "",
          group: null,
          description: "Inlet pressure",
        },
      },
    });
    expect(createOf(create)?.description).toEqual("Inlet pressure");
  });

  it("should parse a channel whose description the Core took: 1,024 code points", () => {
    const description = "\u{1F600}".repeat(1024);
    expect(payloadZ.safeParse({ ...STORED, description }).success).toBe(true);
    expect(new Channel({ ...STORED, description }).description).toEqual(description);
  });

  it("should keep the description in the payload of a channel", () => {
    expect(new Channel(STORED).payload.description).toEqual("Inlet pressure");
  });
});
