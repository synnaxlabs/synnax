// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { Task } from "@/feature/task";

const PREFIXES = ["can", "serial", "tcp", "udp", "arinc429"];

const TASK_TYPES = PREFIXES.flatMap((prefix) => [`${prefix}_read`, `${prefix}_write`]);

describe("bus integration registry", () => {
  it("should register a form for every bus task type", () => {
    expect(Object.keys(Task.FORMS)).toEqual(expect.arrayContaining(TASK_TYPES));
  });

  it("should offer every bus task type in the task selector", () => {
    expect(Task.SELECTABLES.map((s) => s.type)).toEqual(
      expect.arrayContaining(TASK_TYPES),
    );
  });

  it("should register a create command for every bus task type", () => {
    const commands = PREFIXES.flatMap((prefix) => [
      `${prefix}_create_read_task`,
      `${prefix}_create_write_task`,
    ]);
    expect(Task.COMMANDS.map((c) => c.key)).toEqual(expect.arrayContaining(commands));
  });

  it("should register a connect command for every bus device", () => {
    const commands = PREFIXES.map((prefix) => `${prefix}_connect_device`);
    expect(Device.COMMANDS.map((c) => c.key)).toEqual(expect.arrayContaining(commands));
  });

  it("should name bus task types after their integration", () => {
    expect(Task.parseType("can_read")).toBe("CAN read task");
    expect(Task.parseType("serial_write")).toBe("Serial write task");
    expect(Task.parseType("tcp_read")).toBe("TCP read task");
    expect(Task.parseType("udp_write")).toBe("UDP write task");
    expect(Task.parseType("arinc429_read")).toBe("ARINC 429 read task");
  });
});
