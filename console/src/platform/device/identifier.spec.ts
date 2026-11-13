// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Device } from "@/platform/device";
import { createTestDevice } from "@/platform/device/testutil";
import { createAsyncSynnaxWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

describe("identifier", () => {
  let wrapper: React.FC<PropsWithChildren>;

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useIdentifierSuggestions", () => {
    it("should number a suggestion that another device uses", async () => {
      const dev = await createTestDevice(client, { name: uniqueName("Zq Wv") });
      const { result } = renderHook(
        () => Device.useIdentifierSuggestions(dev.name, dev.key),
        { wrapper },
      );
      await waitFor(() => expect(result.current.length).toBeGreaterThan(0));
      const [best] = result.current;
      await createTestDevice(client, { properties: { identifier: best } });
      await waitFor(() => expect(result.current).not.toContain(best));
      expect(result.current[0]).toMatch(new RegExp(`^${best}_\\d+$`));
    });

    it("should keep the identifier of the device itself", async () => {
      const name = uniqueName("Zq Wv");
      const probe = renderHook(() => Device.useIdentifierSuggestions(name, ""), {
        wrapper,
      });
      await waitFor(() => expect(probe.result.current.length).toBeGreaterThan(0));
      const [best] = probe.result.current;
      const dev = await createTestDevice(client, {
        name,
        properties: { identifier: best },
      });
      const { result } = renderHook(
        () => Device.useIdentifierSuggestions(name, dev.key),
        { wrapper },
      );
      await waitFor(() => expect(result.current[0]).toBe(best));
    });
  });

  describe("useConfigureMany", () => {
    it("should give devices that share a name different identifiers", async () => {
      const name = uniqueName("Zq Wv");
      const devices = [
        await createTestDevice(client, { name, configured: false }),
        await createTestDevice(client, { name, configured: false }),
      ];
      const keys = devices.map((d) => d.key);
      const { result } = renderHook(() => Device.useConfigureMany(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ keys, zeroProperties: { rate: 10 } });
      });
      expect(result.current.variant).toBe("success");
      const saved = await client.devices.retrieve({ key: keys });
      expect(saved.map((d) => d.configured)).toEqual([true, true]);
      expect(saved.map((d) => d.properties.rate)).toEqual([10, 10]);
      expect(new Set(saved.map((d) => d.properties.identifier)).size).toBe(2);
    });

    it("should fail for a name that gives no identifier", async () => {
      const dev = await createTestDevice(client, { name: "6xSTG", configured: false });
      const { result } = renderHook(() => Device.useConfigureMany(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ keys: [dev.key], zeroProperties: {} });
      });
      expect(result.current.status.description).toBe(
        "Synnax cannot suggest a free identifier for 6xSTG. Configure the device by " +
          "hand.",
      );
      expect((await client.devices.retrieve(dev.key)).configured).toBe(false);
    });

    it("should leave a configured device as it is", async () => {
      const dev = await createTestDevice(client, {
        properties: { identifier: "kept" },
      });
      const { result } = renderHook(() => Device.useConfigureMany(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ keys: [dev.key], zeroProperties: {} });
      });
      const saved = await client.devices.retrieve(dev.key);
      expect(saved.properties.identifier).toBe("kept");
    });
  });
});
