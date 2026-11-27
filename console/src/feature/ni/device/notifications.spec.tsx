// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { id } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { CaptureStatuses, createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const Listener = (): null => {
  NI.Device.useListenForChanges();
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

describe("NI.Device.useListenForChanges", async () => {
  const rack = await client.racks.create({ name: uniqueName("ni_rack") });
  const create = async (model: string, identifier: string) =>
    await client.ni.devices.create({
      rack: rack.key,
      name: uniqueName("ni_dev"),
      resourceName: id.create(),
      model,
      identifier,
    });

  it("should notify once when an unconfigured device connects", async () => {
    const { statuses } = await renderListener();
    const model = uniqueName("model");
    const dev = await create(model, "");
    await waitFor(() =>
      expect(statuses().find((s) => s.message === `${model} connected`)).toBeDefined(),
    );
    await client.ni.devices.update(dev.key, {
      ...ni.device.configurationZ.parse(dev),
      name: uniqueName("renamed"),
      group: "configuration",
    });
    const other = uniqueName("model");
    await create(other, "");
    await waitFor(() =>
      expect(statuses().find((s) => s.message === `${other} connected`)).toBeDefined(),
    );
    const first = statuses().find((s) => s.message === `${model} connected`);
    expect(first?.count).toBe(1);
    expect(first?.name).toBe(dev.name);
  });

  it("should not notify for a device that has an identifier", async () => {
    const { statuses } = await renderListener();
    const configured = uniqueName("model");
    await create(configured, "dev1");
    const model = uniqueName("model");
    await create(model, "");
    await waitFor(() =>
      expect(statuses().find((s) => s.message === `${model} connected`)).toBeDefined(),
    );
    expect(
      statuses().find((s) => s.message === `${configured} connected`),
    ).toBeUndefined();
  });
});
