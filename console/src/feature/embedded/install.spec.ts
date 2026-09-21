// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import { afterEach, describe, expect, it } from "vitest";

import { Embedded } from "@/feature/embedded";
import { clearSupervisor, mockSupervisor } from "@/feature/embedded/testutil";

describe("Embedded.installMiddleware", () => {
  afterEach(clearSupervisor);

  it("should stop the Core before the install runs", async () => {
    const { commands } = mockSupervisor(() => ({ state: "starting" }));
    await Embedded.installMiddleware(async () => {
      expect(commands.mock.calls).toEqual([["supervisor_stop"]]);
    });
    expect(commands.mock.calls).toEqual([["supervisor_stop"]]);
  });

  it("should start the Core again when the install fails", async () => {
    const { commands } = mockSupervisor(() => ({ state: "starting" }));
    const failure = new Error("the installer failed");
    await expect(
      Embedded.installMiddleware(async () => await Promise.reject(failure)),
    ).rejects.toBe(failure);
    expect(commands.mock.calls).toEqual([["supervisor_stop"], ["supervisor_restart"]]);
  });
});
