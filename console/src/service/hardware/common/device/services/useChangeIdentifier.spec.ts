// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { useChangeIdentifier } from "@/hardware/common/device/services/useChangeIdentifier";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("useChangeIdentifier", () => {
  let wrapper: React.FC<PropsWithChildren>;
  let rack: { key: number };

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    rack = await client.racks.create({ name: `test-rack-${id.create()}` });
  });

  it("should change the identifier on a device", async () => {
    const dev = await client.devices.create({
      key: id.create(),
      name: "Test Device",
      rack: rack.key,
      location: "test-location",
      make: "test-make",
      model: "test-model",
      properties: { identifier: "old-id" },
    });

    const { result } = renderHook(() => useChangeIdentifier(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: dev.key, identifier: "new-id" });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.devices.retrieve({ key: dev.key });
    expect(updated.properties.identifier).toEqual("new-id");
  });

  it("should preserve other properties when changing identifier", async () => {
    const dev = await client.devices.create({
      key: id.create(),
      name: "Test Device",
      rack: rack.key,
      location: "test-location",
      make: "test-make",
      model: "test-model",
      properties: { identifier: "old-id", rate: 100, channels: [1, 2, 3] },
    });

    const { result } = renderHook(() => useChangeIdentifier(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: dev.key, identifier: "new-id" });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.devices.retrieve({ key: dev.key });
    expect(updated.properties.identifier).toEqual("new-id");
    expect(updated.properties.rate).toEqual(100);
    expect(updated.properties.channels).toEqual([1, 2, 3]);
  });

  it("should set identifier on a device with no existing identifier", async () => {
    const dev = await client.devices.create({
      key: id.create(),
      name: "Test Device",
      rack: rack.key,
      location: "test-location",
      make: "test-make",
      model: "test-model",
      properties: { rate: 50 },
    });

    const { result } = renderHook(() => useChangeIdentifier(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: dev.key, identifier: "brand-new" });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.devices.retrieve({ key: dev.key });
    expect(updated.properties.identifier).toEqual("brand-new");
    expect(updated.properties.rate).toEqual(50);
  });
});
