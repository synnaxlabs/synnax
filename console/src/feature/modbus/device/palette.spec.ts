// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { assertDefined, renderHookWithConsole } from "@/testutil";

const client = createTestClient();

describe("Modbus.Device.COMMANDS", () => {
  it("should expose a connect command visible to device creators", async () => {
    const [connect] = Modbus.Device.COMMANDS;
    expect(connect.key).toBe("modbus_connect_server");
    expect(connect.commandName).toBe("Connect a Modbus server");
    assertDefined(connect.useVisible);
    const { result } = await renderHookWithConsole(connect.useVisible, { client });
    await waitFor(() => expect(result.current).toBe(true));
  });
});
