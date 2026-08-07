// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type record } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Device } from "@/platform/device";
import { createTestDevice } from "@/platform/device/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();

describe("useChangeIdentifier", () => {
  let wrapper: React.FC<PropsWithChildren>;

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const changeIdentifier = async (
    dev: device.Device,
    identifier: string,
  ): Promise<record.Unknown> => {
    const { result } = renderHook(() => Device.useChangeIdentifier(), { wrapper });
    await act(async () => {
      await result.current.updateAsync({ key: dev.key, identifier });
    });
    await waitFor(() => expect(result.current.variant).toEqual("success"));
    const updated = await client.devices.retrieve(dev.key);
    return updated.properties;
  };

  it("should change the identifier on a device", async () => {
    const dev = await createTestDevice(client, {
      properties: { identifier: "old_id" },
    });
    const properties = await changeIdentifier(dev, "new_id");
    expect(properties.identifier).toEqual("new_id");
  });

  it("should preserve other properties when changing identifier", async () => {
    const dev = await createTestDevice(client, {
      properties: { identifier: "old_id", rate: 100, channels: [1, 2, 3] },
    });
    const properties = await changeIdentifier(dev, "new_id");
    expect(properties.identifier).toEqual("new_id");
    expect(properties.rate).toEqual(100);
    expect(properties.channels).toEqual([1, 2, 3]);
  });

  it("should set identifier on a device with no existing identifier", async () => {
    const dev = await createTestDevice(client, { properties: { rate: 50 } });
    const properties = await changeIdentifier(dev, "brand_new");
    expect(properties.identifier).toEqual("brand_new");
    expect(properties.rate).toEqual(50);
  });
});
