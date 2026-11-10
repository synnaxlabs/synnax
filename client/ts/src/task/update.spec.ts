// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { task } from "@/task";
import { reduceAll } from "@/task/actions";

const configZ = z.object({ rate: z.number() });

describe("Update", () => {
  it("should hold the fields that only the Core sets", () => {
    const update: task.Update = { configHash: "h1" };
    expect(task.updateZ().parse(update)).toEqual(update);
  });

  it("should replace the config whole with the config type of the task", () => {
    const update: task.Update<typeof configZ> = { config: { rate: 10 } };
    expect(task.updateZ(configZ).parse(update)).toEqual(update);
  });

  it("should leave an absent config absent", () => {
    expect(task.updateZ().parse({ name: "Read" })).toEqual({ name: "Read" });
  });
});

describe("UpdateInput", () => {
  it("should leave out the fields that only the Core sets", () => {
    // @ts-expect-error - only the Core sets configHash.
    const input: task.UpdateInput = { configHash: "h1" };
    expect(input).toEqual({ configHash: "h1" });
  });

  it("should check the config against the config type of the task", () => {
    const input: task.UpdateInput<typeof configZ> = { config: { rate: 10 } };
    // @ts-expect-error - the config type has no field name.
    const bad: task.UpdateInput<typeof configZ> = { config: { name: "x" } };
    expect(task.updateZ(configZ).parse(input)).toEqual(input);
    expect(task.updateZ(configZ).safeParse(bad).success).toBe(false);
  });
});

describe("reduceAll", () => {
  it("should replace the config whole", () => {
    const prev = task.payloadZ().parse({
      key: "00000000-0000-4000-8000-000000000004",
      name: "Read",
      type: "ni_analog_read",
      config: { rate: 10, device: "d" },
    });
    const { next, inverse } = reduceAll(prev, [
      task.update({ value: { config: { rate: 20 } } }),
    ]);
    expect(next.config).toEqual({ rate: 20 });
    expect(reduceAll(next, inverse).next).toEqual(prev);
  });
});
