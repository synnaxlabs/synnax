// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { patch } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { rack } from "@/rack";
import { reduceAll } from "@/rack/actions";

describe("Update", () => {
  it("should take null on an optional field to clear it", () => {
    const update: rack.Update = { version: null, group: null };
    expect(rack.updateZ.parse(update)).toEqual(update);
  });

  it("should reject null on a required field", () => {
    // @ts-expect-error - only an optional field takes null.
    const update: rack.Update = { name: null };
    expect(rack.updateZ.safeParse(update).success).toBe(false);
  });

  it("should take operators on a list", () => {
    const update: rack.Update = { integrations: { insert: ["labjack"] } };
    expect(rack.updateZ.parse(update)).toEqual(update);
  });

  it("should reject a plain value on a list", () => {
    // @ts-expect-error - a list takes operators.
    const update: rack.Update = { integrations: ["labjack"] };
    expect(rack.updateZ.safeParse(update).success).toBe(false);
  });
});

describe("reduceAll", () => {
  const createRack = (): rack.Payload =>
    rack.payloadZ.parse({
      key: 12,
      name: "Edge",
      integrations: ["ni"],
      version: "1.0.0",
      role: "00000000-0000-4000-8000-000000000001",
    });

  it("should apply an update and return the update that reverts it", () => {
    const prev = createRack();
    const { next, inverse, targets } = reduceAll(prev, [
      rack.update({
        value: { integrations: { insert: ["labjack"] }, version: null },
      }),
    ]);
    expect(next.version).toBeUndefined();
    expect(next.integrations).toEqual(["ni", "labjack"]);
    expect(targets).toEqual(["integrations", "version"]);
    expect(reduceAll(next, inverse).next).toEqual(prev);
  });

  it("should set a required field", () => {
    const { next } = reduceAll(createRack(), [rack.update({ value: { name: "Lab" } })]);
    expect(next.name).toBe("Lab");
  });

  it("should throw a stale error when a guard does not hold", () => {
    expect(() =>
      reduceAll(createRack(), [
        rack.update({
          value: { version: null },
          guards: [{ path: ["name"], value: "Other" }],
        }),
      ]),
    ).toThrow(patch.StaleError);
  });
});
