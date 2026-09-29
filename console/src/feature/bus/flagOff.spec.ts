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

vi.mock("@/flags", () => ({ FLAGS: { library: true, can: false } }));

describe("bus integration flag off", () => {
  it("should hide every bus form, selectable, and command when off", () => {
    const types = Object.keys(Task.FORMS);
    BUS_TASK_TYPES.forEach((type) => expect(types).not.toContain(type));
    const selectable = Task.SELECTABLES.map((s) => s.type);
    BUS_TASK_TYPES.forEach((type) => expect(selectable).not.toContain(type));
    const commands = [...Task.COMMANDS, ...Device.COMMANDS].map((c) => c.key);
    [...BUS_TASK_COMMANDS, ...BUS_CONNECT_COMMANDS].forEach((key) =>
      expect(commands).not.toContain(key),
    );
    expect(Task.parseType("can_read")).toBe("Can read task");
    expect(Task.parseType("mil1553_write")).toBe("Mil1553 write task");
    expect(Object.keys(Task.FORMS)).toContain("modbus_read");
  });
});
