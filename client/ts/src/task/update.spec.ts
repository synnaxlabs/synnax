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

import { type ni } from "@/ni";
import { task } from "@/task";

const configZ = z.object({ rate: z.number(), device: z.string() });
const configUpdateZ = z.object({
  rate: z.number().optional(),
  device: z.string().optional(),
});
const configSchemas = { u: configUpdateZ, t: configZ };

describe("Update", () => {
  it("should hold the fields that only the Core sets", () => {
    const update: task.Update = { configHash: "h1" };
    expect(task.updateZ().parse(update)).toEqual(update);
  });

  it("should take a partial config with the schemas of the config", () => {
    const update: task.Update<typeof configSchemas> = { config: { rate: 10 } };
    expect(task.updateZ(configSchemas).parse(update)).toEqual(update);
  });

  it("should take a whole config without the schemas of the config", () => {
    const update: task.Update = { config: { anything: 1 } };
    expect(task.updateZ().parse(update)).toEqual(update);
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

  it("should check the config against the schemas of the config", () => {
    const input: task.UpdateInput<typeof configSchemas> = { config: { rate: 10 } };
    // @ts-expect-error - the rate of the config is a number.
    const bad: task.UpdateInput<typeof configSchemas> = { config: { rate: "x" } };
    expect(task.updateZ(configSchemas).parse(input)).toEqual(input);
    expect(task.updateZ(configSchemas).safeParse(bad).success).toBe(false);
  });
});

describe("Client.update", () => {
  it("should not take a field that only the Core sets", () => {
    const input: Parameters<task.Client["update"]>[0] = {
      key: "00000000-0000-4000-8000-000000000004",
      // @ts-expect-error - only the Core sets configHash.
      configHash: "h1",
    };
    expect(input).toHaveProperty("configHash", "h1");
  });

  it("should check the config against the schemas that the caller names", () => {
    type Scan = (typeof ni.CONFIG_SCHEMAS)[typeof ni.SCANNER_TASK_TYPE];
    const update = async (tasks: task.Client) =>
      await tasks.update<Scan>({
        key: "00000000-0000-4000-8000-000000000004",
        // @ts-expect-error - disabled is a boolean.
        config: { disabled: "no" },
      });
    expect(update).toBeTypeOf("function");
  });
});

describe("createReducer", () => {
  const reduceAll = task.createReducer({ ni_analog_read: configSchemas });
  const createTask = (type: string): task.Payload =>
    task.payloadZ().parse({
      key: "00000000-0000-4000-8000-000000000004",
      name: "Read",
      type,
      config: { rate: 10, device: "d" },
    });

  it("should merge the config by the schemas of the task type", () => {
    const prev = createTask("ni_analog_read");
    const { next, inverse } = reduceAll(prev, [
      task.update({ value: { config: { rate: 20 } } }),
    ]);
    expect(next.config).toEqual({ rate: 20, device: "d" });
    expect(reduceAll(next, inverse).next).toEqual(prev);
  });

  it("should replace the config when the update sets it", () => {
    const prev = createTask("ni_analog_read");
    const { next } = reduceAll(prev, [
      task.update({ value: { config: { set: { rate: 20, device: "e" } } } }),
    ]);
    expect(next.config).toEqual({ rate: 20, device: "e" });
  });

  it("should replace the config whole when the task type has no schemas", () => {
    const prev = createTask("arc");
    const { next, inverse } = reduceAll(prev, [
      task.update({ value: { config: { rate: 20 } } }),
    ]);
    expect(next.config).toEqual({ rate: 20 });
    expect(reduceAll(next, inverse).next).toEqual(prev);
  });
});
