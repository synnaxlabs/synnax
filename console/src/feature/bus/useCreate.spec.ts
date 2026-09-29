// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { can } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { renderHook } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { createBusDevice, createBusLibrary } from "@/feature/bus/testutil";
import { CAN } from "@/feature/can";
import { awaitTaskResourceTab, createSelectedPanel } from "@/platform/task/testutil";
import { createConsoleWrapper, createTestStore } from "@/testutil";

const client = createTestClient();

const setup = async () => {
  const store = await createTestStore();
  const { wrapper } = await createConsoleWrapper({ client, store });
  const created = await createSelectedPanel(store, client);
  const { result } = renderHook(() => CAN.Task.useCreateRead(), { wrapper });
  return { created, result };
};

describe("bus task create", () => {
  it("should create a draft on an existing library and open it", async () => {
    await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      CAN.Device.MAKE,
      can.propertiesZ.parse({}),
    );
    const { created, result } = await setup();
    act(() => result.current({ deviceKey: dev.key }));
    const key = await awaitTaskResourceTab(created);
    const tsk = await client.tasks.retrieve({ key, schemas: CAN.Task.READ_SCHEMAS });
    expect(tsk.name).toBe("CAN read task");
    expect(tsk.config.device).toBe(dev.key);
    const lib = await client.libraries.retrieve({ key: tsk.config.library });
    expect(lib.key).toBe(tsk.config.library);
  });

  it("should create a draft on the library the caller names", async () => {
    const { library } = await createBusLibrary(client);
    const { created, result } = await setup();
    act(() => result.current({ config: { library: library.key } }));
    const key = await awaitTaskResourceTab(created);
    const tsk = await client.tasks.retrieve({ key, schemas: CAN.Task.READ_SCHEMAS });
    expect(tsk.config.library).toBe(library.key);
  });
});
