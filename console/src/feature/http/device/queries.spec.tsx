// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { describe, it } from "vitest";

import { HTTP } from "@/feature/http";
import { createHTTPDevice } from "@/feature/http/testutil";
import { renderWithDeviceForm } from "@/platform/device/testutil";

const client = createTestClient();

const DeviceTimeout = () => (
  <span>{`timeout:${HTTP.Device.useFromConfig()?.properties.timeoutMs}`}</span>
);

describe("useFromConfig", () => {
  it("should return the device the form's config names", async () => {
    const dev = await createHTTPDevice(client, { properties: { timeoutMs: 1234 } });
    await renderWithDeviceForm(<DeviceTimeout />, { deviceKey: dev.key, client });
    await screen.findByText("timeout:1234");
  });
});
