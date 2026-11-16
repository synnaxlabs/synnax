// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { afterEach, describe, expect, it } from "vitest";

import { Embedded } from "@/feature/embedded";
import { clearSupervisor, mockSupervisor, RUNNING } from "@/feature/embedded/testutil";

describe("Embedded.beforeClose", () => {
  afterEach(clearSupervisor);

  it("should quit through the supervisor when the main window closes", async () => {
    const { commands } = mockSupervisor(() => RUNNING);
    await Embedded.beforeClose(Drift.MAIN_WINDOW);
    expect(commands).toHaveBeenCalledWith("supervisor_quit");
  });

  it("should close any other window without a quit", async () => {
    const { commands } = mockSupervisor(() => RUNNING);
    await Embedded.beforeClose("child");
    expect(commands).not.toHaveBeenCalled();
  });
});
