// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import {
  BUS_CONNECT_COMMANDS,
  BUS_TASK_COMMANDS,
  BUS_TASK_TYPES,
} from "@/feature/bus/testutil";
import { Device } from "@/feature/device";
import { Task } from "@/feature/task";

vi.mock("@/flags", () => ({ FLAGS: { library: true, can: true } }));

describe("bus integration flag on", () => {
  it("should register every bus form, selectable, and command when on", () => {
    expect(Object.keys(Task.FORMS)).toEqual(expect.arrayContaining(BUS_TASK_TYPES));
    expect(Task.SELECTABLES.map((s) => s.type)).toEqual(
      expect.arrayContaining(BUS_TASK_TYPES),
    );
    expect(Task.COMMANDS.map((c) => c.key)).toEqual(
      expect.arrayContaining(BUS_TASK_COMMANDS),
    );
    expect(Device.COMMANDS.map((c) => c.key)).toEqual(
      expect.arrayContaining(BUS_CONNECT_COMMANDS),
    );
    expect(Task.parseType("can_read")).toBe("CAN read task");
    expect(Task.parseType("arinc429_read")).toBe("ARINC 429 read task");
    expect(Task.parseType("mil1553_write")).toBe("MIL-STD-1553 write task");
  });
});
