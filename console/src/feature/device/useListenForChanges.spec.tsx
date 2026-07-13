// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/pluto";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { createTestDevice } from "@/platform/device/testutil";
import { CaptureStatuses, createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const Listener = (): null => {
  Device.useListenForChanges();
  return null;
};

const renderListener = async (): Promise<{
  statuses: () => Status.NotificationSpec[];
}> => {
  const { wrapper } = await createConsoleWrapper({ client });
  let latest: Status.NotificationSpec[] = [];
  render(
    <>
      <Listener />
      <CaptureStatuses onStatuses={(s) => (latest = s)} />
    </>,
    { wrapper },
  );
  return { statuses: () => latest };
};

describe("device/useListenForChanges", () => {
  it("notifies once when a new unconfigured device connects", async () => {
    const { statuses } = await renderListener();
    const model = uniqueName("model");
    const dev = await createTestDevice(client, { configured: false, model });
    await waitFor(() =>
      expect(
        statuses().find((s) => s.message === `New ${model} connected`),
      ).toBeDefined(),
    );
    await client.devices.create(dev);
    const other = uniqueName("model");
    await createTestDevice(client, { configured: false, model: other });
    await waitFor(() =>
      expect(
        statuses().find((s) => s.message === `New ${other} connected`),
      ).toBeDefined(),
    );
    const first = statuses().find((s) => s.message === `New ${model} connected`);
    expect(first?.count).toBe(1);
  });

  it("does not notify for devices that are already configured", async () => {
    const { statuses } = await renderListener();
    const configuredModel = uniqueName("model");
    await createTestDevice(client, { configured: true, model: configuredModel });
    const model = uniqueName("model");
    await createTestDevice(client, { configured: false, model });
    await waitFor(() =>
      expect(
        statuses().find((s) => s.message === `New ${model} connected`),
      ).toBeDefined(),
    );
    expect(
      statuses().find((s) => s.message === `New ${configuredModel} connected`),
    ).toBeUndefined();
  });
});
