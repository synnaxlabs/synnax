// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, rack } from "@synnaxlabs/client";
import { id, status } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Rack } from "@/hardware/rack";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should return a list of rack keys", async () => {
      const rack1 = await client.hardware.racks.create({
        name: "rack1",
      });
      const rack2 = await client.hardware.racks.create({
        name: "rack2",
      });

      const { result } = renderHook(() => Rack.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(rack1.key);
      expect(result.current.data).toContain(rack2.key);
    });

    it("should get individual racks using getItem", async () => {
      const testRack = await client.hardware.racks.create({
        name: "testRack",
      });

      const { result } = renderHook(() => Rack.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedRack = result.current.getItem(testRack.key);
      expect(retrievedRack?.key).toEqual(testRack.key);
      expect(retrievedRack?.name).toEqual("testRack");
    });

    it("should filter racks by search term", async () => {
      await client.hardware.racks.create({
        name: "ordinary",
      });
      await client.hardware.racks.create({
        name: "special",
      });

      const { result } = renderHook(() => Rack.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ searchTerm: "special" });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(1);
      expect(
        result.current.data
          .map((key: rack.Key) => result.current.getItem(key)?.name)
          .includes("special"),
      ).toBe(true);
    });

    it("should handle pagination with limit and offset", async () => {
      for (let i = 0; i < 5; i++)
        await client.hardware.racks.create({
          name: `paginationRack${i}`,
        });

      const { result } = renderHook(() => Rack.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
    });

    it("should include status when requested", async () => {
      const testRack = await client.hardware.racks.create({
        name: "statusRack",
      });

      const { result } = renderHook(() => Rack.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ includeStatus: true });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrievedRack = result.current.getItem(testRack.key);
      expect(retrievedRack?.key).toEqual(testRack.key);
    });

    it("should update the list when a rack is created", async () => {
      const { result } = renderHook(() => Rack.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      const initialLength = result.current.data.length;

      const newRack = await client.hardware.racks.create({
        name: "newRack",
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThan(initialLength);
        expect(result.current.data).toContain(newRack.key);
      });
    });

    it("should update the list when a rack is updated", async () => {
      const testRack = await client.hardware.racks.create({
        name: "original",
      });

      const { result } = renderHook(() => Rack.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.getItem(testRack.key)?.name).toEqual("original");

      await client.hardware.racks.create({
        ...testRack,
        name: "updated",
      });

      await waitFor(() => {
        expect(result.current.getItem(testRack.key)?.name).toEqual("updated");
      });
    });

    it("should remove rack from list when deleted", async () => {
      const testRack = await client.hardware.racks.create({
        name: "toDelete",
      });

      const { result } = renderHook(() => Rack.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data).toContain(testRack.key);

      await client.hardware.racks.delete(testRack.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testRack.key);
      });
    });

    it("should update rack status in the list", async () => {
      const testRack = await client.hardware.racks.create({
        name: "statusRack",
      });

      const { result } = renderHook(() => Rack.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const rackStatus: rack.Status = status.create<typeof rack.statusDetailsSchema>({
        key: id.create(),
        variant: "warning",
        message: "Rack needs attention",
        details: {
          rack: testRack.key,
        },
      });

      await act(async () => {
        const writer = await client.openWriter([rack.STATUS_CHANNEL_NAME]);
        await writer.write(rack.STATUS_CHANNEL_NAME, [rackStatus]);
        await writer.close();
      });

      await waitFor(() => {
        const rackInList = result.current.getItem(testRack.key);
        expect(rackInList?.status?.variant).toEqual("warning");
        expect(rackInList?.status?.message).toEqual("Rack needs attention");
      });
    });
  });

  describe("useDelete", () => {
    it("should delete a single rack", async () => {
      const testRack = await client.hardware.racks.create({
        name: "delete_single",
      });
      const { result } = renderHook(
        () => {
          const list = Rack.useList();
          const del = Rack.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => expect(result.current.list.variant).toEqual("success"));
      expect(result.current.list.data).toContain(testRack.key);

      await act(async () => {
        await result.current.del.updateAsync(testRack.key);
      });
      await waitFor(() => {
        expect(result.current.list.data).not.toContain(testRack.key);
      });
    });

    it("should delete multiple racks", async () => {
      const rack1 = await client.hardware.racks.create({
        name: "delete_multi_1",
      });
      const rack2 = await client.hardware.racks.create({
        name: "delete_multi_2",
      });
      const { result } = renderHook(
        () => {
          const list = Rack.useList();
          const del = Rack.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => expect(result.current.list.variant).toEqual("success"));
      expect(result.current.list.data).toContain(rack1.key);
      expect(result.current.list.data).toContain(rack2.key);

      await act(async () => {
        await result.current.del.updateAsync([rack1.key, rack2.key]);
      });
      await waitFor(() => {
        expect(result.current.list.data).not.toContain(rack1.key);
        expect(result.current.list.data).not.toContain(rack2.key);
      });
    });
  });

  describe("useRename", () => {
    it("should rename a rack", async () => {
      const testRack = await client.hardware.racks.create({
        name: "original_name",
      });
      const { result } = renderHook(
        () => {
          const retrieve = Rack.useRetrieve({ key: testRack.key });
          const rename = Rack.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      expect(result.current.retrieve.data?.name).toEqual("original_name");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: testRack.key,
          name: "renamed_rack",
        });
      });
      await waitFor(() => {
        expect(result.current.retrieve.data?.name).toEqual("renamed_rack");
      });
    });

    it("should update rack in list after rename", async () => {
      const testRack = await client.hardware.racks.create({
        name: "list_original",
      });
      const { result } = renderHook(
        () => {
          const list = Rack.useList();
          const rename = Rack.useRename();
          return { list, rename };
        },
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => expect(result.current.list.variant).toEqual("success"));
      expect(result.current.list.getItem(testRack.key)?.name).toEqual("list_original");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: testRack.key,
          name: "list_renamed",
        });
      });
      await waitFor(() => {
        expect(result.current.list.getItem(testRack.key)?.name).toEqual("list_renamed");
      });
    });
  });
});
