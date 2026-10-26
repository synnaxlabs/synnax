// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import {
  BUS_CONNECT_COMMANDS,
  BUS_TASK_COMMANDS,
  BUS_TASK_TYPES,
} from "@/feature/bus/testutil";
import { Device } from "@/feature/device";
import { Task } from "@/feature/task";
import { renderModalOpener } from "@/platform/modals/testutil";

vi.mock("@/flags", () => ({ FLAGS: { icd: true, can: true } }));

describe("bus integration flag on", () => {
  it("should register every bus selectable and command when on", () => {
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
  });

  it("should list every bus integration in the device picker when on", async () => {
    await renderModalOpener(Device.useConnectModal, [{}]);
    for (const title of ["CAN device", "serial device", "TCP device", "UDP device"])
      expect(await screen.findByText(title)).toBeTruthy();
  });
});
