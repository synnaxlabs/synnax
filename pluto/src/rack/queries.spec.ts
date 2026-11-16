// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack, status } from "@synnaxlabs/client";
import { createTestClient, spyOnSend } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Rack } from "@/rack";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("use", () => {
    it("should retrieve a rack by its key", async () => {
      const testRack = await client.racks.create({ name: "testRack" });
      const { result } = await renderHookSuspended(
        () => Rack.use({ key: testRack.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.key).toEqual(testRack.key);
      expect(result.current?.name).toEqual("testRack");
    });

    it("should include the status of the rack by default", async () => {
      const testRack = await client.racks.create({ name: "testRack" });
      const { result } = await renderHookSuspended(
        () => Rack.use({ key: testRack.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.status).toBeDefined();
      expect(result.current?.status?.time.nanoseconds).toBeGreaterThan(0);
    });

    it("should include the status of the rack even when the query is cached", async () => {
      const testRack = await client.racks.create({ name: "testRack2" });
      const { result: result1 } = await renderHookSuspended(
        () => Rack.use({ key: testRack.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current).not.toBeNull());
      expect(result1.current?.status).toBeDefined();
      expect(result1.current?.status?.time.nanoseconds).toBeGreaterThan(0);
      const rackStatus: rack.Status = status.create<typeof rack.statusDetailsZ>({
        owner: rack.resourceID(testRack.key),
        variant: "success",
        message: "Rack is happy as a clam",
        details: { running: true },
      });
      await act(async () => {
        await client.statuses.set(rackStatus);
      });
      const { result: result2 } = await renderHookSuspended(
        () => Rack.use({ key: testRack.key }),
        { wrapper },
      );
      await waitFor(() => expect(result2.current).not.toBeNull());
      await waitFor(() => {
        expect(result2.current?.status?.variant).toEqual("success");
        expect(result2.current?.status?.message).toEqual("Rack is happy as a clam");
      });
    });

    describe("rack cached without a status", () => {
      it("should fetch the status", async () => {
        const testRack = await client.racks.create({ name: "status-free-read" });
        const local = createTestClient();
        await local.racks.retrieve(testRack.key);
        const { result } = await renderHookSuspended(
          () => Rack.use({ key: testRack.key }),
          { wrapper: await createAsyncSynnaxWrapper({ client: local }) },
        );
        expect(result.current?.status?.owner).toEqual(rack.resourceID(testRack.key));
      });

      it("should fetch once when the rack has no status row", async () => {
        const testRack = await client.racks.create({ name: "no-status-row" });
        const [owned] = await client.statuses.retrieve({
          owners: [rack.resourceID(testRack.key)],
        });
        await client.statuses.delete(owned.key);
        const local = createTestClient();
        await local.racks.retrieve(testRack.key);
        const localWrapper = await createAsyncSynnaxWrapper({ client: local });
        const send = spyOnSend(local);
        const render = async () =>
          await renderHookSuspended(() => Rack.use({ key: testRack.key }), {
            wrapper: localWrapper,
          });
        const first = await render();
        expect(first.result.current?.status).toBeUndefined();
        first.unmount();
        const second = await render();
        expect(second.result.current?.key).toEqual(testRack.key);
        const retrieves = send.mock.calls.filter(
          ([target]) => target === "/rack/retrieve",
        );
        expect(retrieves).toHaveLength(1);
      });
    });
  });

  describe("useList", () => {
    const setStatus = async (key: rack.Key) =>
      await client.statuses.set(
        status.create({
          owner: rack.resourceID(key),
          variant: "warning",
          message: "Rack needs attention",
        }),
      );

    it("should update the status of a rack in the list", async () => {
      const r = await client.racks.create({ name: id.create() });
      const { result } = renderHook(
        () => Rack.useList({ query: { where: { name: { eq: r.name } } } }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([r.key]));
      await setStatus(r.key);
      await waitFor(() =>
        expect(result.current.getItem(r.key)?.status?.variant).toEqual("warning"),
      );
    });

    it("should read the status of a rack outside the list", async () => {
      const r = await client.racks.create({ name: id.create() });
      await setStatus(r.key);
      const missing = id.create();
      const { result } = renderHook(
        () => Rack.useList({ query: { where: { name: { eq: missing } } } }),
        { wrapper },
      );
      const callback = vi.fn();
      const release = result.current.subscribe(callback, r.key);
      await waitFor(() => expect(callback).toHaveBeenCalled());
      expect(result.current.getItem(r.key)?.status?.variant).toEqual("warning");
      release();
    });
  });

  describe("useMultiple", () => {
    it("should retrieve only the racks with the integration", async () => {
      const integration = id.create();
      const match = await client.racks.create({
        name: "match",
        integrations: [integration],
      });
      await client.racks.create({ name: "other" });
      const { result } = await renderHookSuspended(
        () => Rack.useMultiple({ where: { integrations: { any: [integration] } } }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current?.map((r) => r.key)).toEqual([match.key]),
      );
    });

    it("should add a matching rack created after the query resolves", async () => {
      const integration = id.create();
      const first = await client.racks.create({
        name: "first",
        integrations: [integration],
      });
      const { result } = await renderHookSuspended(
        () => Rack.useMultiple({ where: { integrations: { any: [integration] } } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).toHaveLength(1));
      const second = await client.racks.create({
        name: "second",
        integrations: [integration],
      });
      await waitFor(() =>
        expect(result.current?.map((r) => r.key).sort()).toEqual(
          [first.key, second.key].sort(),
        ),
      );
    });
  });

  describe("useDelete", () => {
    it("should delete a single rack", async () => {
      const testRack = await client.racks.create({
        name: "delete_single",
      });
      const { result } = renderHook(
        () => {
          const list = Rack.useList({
            query: { where: { key: { in: [testRack.key] } } },
          });
          const del = Rack.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => result.current.list.more());
      await waitFor(() => expect(result.current.list.data).toEqual([testRack.key]));

      await act(async () => {
        await result.current.del.updateAsync(testRack.key);
      });
      await waitFor(() => {
        expect(result.current.list.data).not.toContain(testRack.key);
      });
    });

    it("should delete multiple racks", async () => {
      const rack1 = await client.racks.create({
        name: "delete_multi_1",
      });
      const rack2 = await client.racks.create({
        name: "delete_multi_2",
      });
      const { result } = renderHook(
        () => {
          const list = Rack.useList({
            query: { where: { key: { in: [rack1.key, rack2.key] } } },
          });
          const del = Rack.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => result.current.list.more());
      await waitFor(() => expect(result.current.list.data).toHaveLength(2));

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
      const testRack = await client.racks.create({
        name: "original_name",
      });
      const { result } = await renderHookSuspended(
        () => {
          const retrieve = Rack.use({ key: testRack.key });
          const rename = Rack.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve).not.toBeNull());
      expect(result.current.retrieve?.name).toEqual("original_name");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: testRack.key,
          name: "renamed_rack",
        });
      });
      await waitFor(() => {
        expect(result.current.retrieve?.name).toEqual("renamed_rack");
      });
    });

    it("should update rack in list after rename", async () => {
      const testRack = await client.racks.create({
        name: "list_original",
      });
      const { result } = renderHook(
        () => {
          const list = Rack.useList({
            query: { where: { key: { in: [testRack.key] } } },
          });
          const rename = Rack.useRename();
          return { list, rename };
        },
        { wrapper },
      );
      act(() => result.current.list.more());
      await waitFor(() =>
        expect(result.current.list.getItem(testRack.key)?.name).toEqual(
          "list_original",
        ),
      );

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
