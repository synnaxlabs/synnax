// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { createHTTPDevice } from "@/feature/http/testutil";
import { renderWithDeviceForm } from "@/platform/device/testutil";
import { createAsyncSynnaxWrapper, renderHookSuspended } from "@/testutil";

const client = createTestClient();

const DeviceName = () => (
  <span>{HTTP.Device.useFromConfig()?.name ?? "no device"}</span>
);

describe("HTTP device queries", () => {
  it("should retrieve the device with its typed properties", async () => {
    const dev = await createHTTPDevice(client, { properties: { timeoutMs: 1234 } });
    const wrapper = await createAsyncSynnaxWrapper({ client });
    const { result } = await renderHookSuspended(
      () => HTTP.Device.use({ key: dev.key }),
      { wrapper },
    );
    await waitFor(() => expect(result.current?.properties.timeoutMs).toBe(1234));
  });

  it("should return the device the form's config names", async () => {
    const dev = await createHTTPDevice(client);
    await renderWithDeviceForm(<DeviceName />, { deviceKey: dev.key, client });
    await screen.findByText(dev.name);
  });
});
