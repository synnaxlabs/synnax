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
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { createRunningRack } from "@/platform/device/testutil";
import { renderModalOpener } from "@/platform/modals/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const Connect = ({ rack }: Device.ConnectProps) => <p>{`Connect from ${rack}`}</p>;

describe("createConnectModal", () => {
  it("should start a new connection from the only running rack with the integration", async () => {
    const integration = uniqueName("integration");
    const rack = await createRunningRack(client, integration);
    const useConnectModal = Device.createConnectModal(Connect, integration);
    const { reopen } = await renderModalOpener(useConnectModal, [], { client });
    // The opener reads the rack list when called, so it reopens until the list loads. It
    // polls on an interval, since each reopen changes the DOM.
    await vi.waitFor(() => {
      reopen();
      expect(screen.getByText(`Connect from ${rack.key}`)).toBeTruthy();
    });
  });

  it("should start from the rack the caller names", async () => {
    const integration = uniqueName("integration");
    await createRunningRack(client, integration);
    const named = await client.racks.create({ name: uniqueName("rack") });
    const useConnectModal = Device.createConnectModal(Connect, integration);
    await renderModalOpener(useConnectModal, [{ rack: named.key }], { client });
    expect(await screen.findByText(`Connect from ${named.key}`)).toBeTruthy();
  });
});
