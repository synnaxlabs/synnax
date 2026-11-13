// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ni } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { createAsyncSynnaxWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

describe("NI chassis queries", async () => {
  const rack = await client.racks.create({ name: uniqueName("ni_rack") });
  const createChassis = async (): Promise<ni.chassis.Chassis> =>
    await client.ni.chassis.create({
      rack: rack.key,
      name: uniqueName("ni_chassis"),
      resourceName: id.create(),
      model: "cDAQ-9178",
    });

  describe("useRetrieve", () => {
    it("should retrieve the chassis with its status", async () => {
      const chassis = await createChassis();
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(
        () => NI.Chassis.useRetrieve({ key: chassis.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.data?.name).toBe(chassis.name));
      expect(result.current.data?.status).toBeDefined();
    });
  });

  describe("useList", () => {
    it("should list the chassis on a rack", async () => {
      const chassis = await createChassis();
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(
        () => NI.Chassis.useList({ query: { where: { rack: { eq: rack.key } } } }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toContain(chassis.key));
    });
  });

  describe("useRename", () => {
    it("should rename the chassis", async () => {
      const chassis = await createChassis();
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(() => NI.Chassis.useRename(), { wrapper });
      const name = uniqueName("renamed");
      await act(async () => {
        await result.current.updateAsync({ key: chassis.key, name });
      });
      expect((await client.ni.chassis.retrieve({ key: chassis.key })).name).toBe(name);
    });
  });

  describe("useDelete", () => {
    it("should delete the chassis", async () => {
      const chassis = await createChassis();
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(() => NI.Chassis.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(chassis.key);
      });
      await expect(
        client.ni.chassis.retrieve({ where: { key: { eq: chassis.key } } }),
      ).resolves.toEqual([]);
    });
  });
});
