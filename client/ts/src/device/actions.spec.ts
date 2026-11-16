// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { createOf, reduceAll } from "@/device/actions";
import { create, set, setRack } from "@/device/actions.gen";
import { type Device } from "@/device/types.gen";

const DEVICE: Omit<Device, "status"> = {
  key: "d1",
  rack: 1,
  location: "dev1",
  make: "ni",
  model: "pxi",
  name: "Device",
  configured: false,
  properties: { readIndex: 1 },
  group: "0198b5a4-0000-7000-8000-000000000001",
};

describe("device actions", () => {
  it("should apply a wire-form set inside the properties", () => {
    const { next } = reduceAll(DEVICE, [
      set({ path: ["properties", "read_index"], value: 2 }),
      set({ path: ["properties", "write_state"], value: { state_index: 3 } }),
    ]);
    expect(next.properties).toEqual({ readIndex: 2, writeState: { stateIndex: 3 } });
  });

  it("should move a device to a rack and out of its group", () => {
    const { next } = reduceAll(DEVICE, [setRack({ rack: 2, parent: "d0" })]);
    expect(next).toEqual({ ...DEVICE, rack: 2, parent: "d0", group: undefined });
    expect("group" in next).toBe(false);
  });

  it("should drop the status that a create carries", () => {
    const created = { ...DEVICE, name: "Created", status: undefined };
    const action = create({ device: created });
    expect(createOf(action)).toEqual({ ...DEVICE, name: "Created" });
    expect(reduceAll(DEVICE, [action]).next).toEqual({ ...DEVICE, name: "Created" });
  });
});
