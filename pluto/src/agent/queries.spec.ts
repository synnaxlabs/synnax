// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Agent } from "@/agent";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should return a list of agent keys", async () => {
      const a1 = await client.agents.create({ name: "agent1" });
      const a2 = await client.agents.create({ name: "agent2" });

      const { result } = renderHook(() => Agent.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(a1.key);
      expect(result.current.data).toContain(a2.key);
    });

    it("should get individual agents using getItem", async () => {
      const a = await client.agents.create({ name: "testAgent" });

      const { result } = renderHook(() => Agent.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrieved = result.current.getItem(a.key);
      expect(retrieved?.key).toEqual(a.key);
      expect(retrieved?.name).toEqual("testAgent");
    });

    it("should update the list when an agent is created externally", async () => {
      const { result } = renderHook(() => Agent.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialLength = result.current.data.length;

      const newAgent = await client.agents.create({ name: "externalAgent" });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(initialLength + 1);
        expect(result.current.data).toContain(newAgent.key);
      });
    });

    it("should remove agent from list when deleted", async () => {
      const a = await client.agents.create({ name: "toDelete" });

      const { result } = renderHook(() => Agent.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(a.key);

      await client.agents.delete(a.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(a.key);
      });
    });

    it("should handle multiple agent updates simultaneously", async () => {
      const a1 = await client.agents.create({ name: "multi1" });
      const a2 = await client.agents.create({ name: "multi2" });

      const { result } = renderHook(() => Agent.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(a1.key);
      expect(result.current.data).toContain(a2.key);

      await Promise.all([
        client.agents.delete(a1.key),
        client.agents.delete(a2.key),
      ]);

      await waitFor(() => {
        expect(result.current.data).not.toContain(a1.key);
        expect(result.current.data).not.toContain(a2.key);
      });
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a single agent by key", async () => {
      const a = await client.agents.create({ name: "singleAgent" });

      const { result } = renderHook(() => Agent.useRetrieve({ key: a.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data?.key).toEqual(a.key);
      expect(result.current.data?.name).toEqual("singleAgent");
    });

    it("should auto-update when agent is deleted via broadcast", async () => {
      const a = await client.agents.create({ name: "reactiveAgent" });

      const { result } = renderHook(
        () => ({
          retrieve: Agent.useRetrieve({ key: a.key }),
          list: Agent.useList(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.retrieve.variant).toEqual("success");
        expect(result.current.list.variant).toEqual("success");
      });
      expect(result.current.retrieve.data?.key).toEqual(a.key);
      expect(result.current.list.data).toContain(a.key);

      await client.agents.delete(a.key);

      await waitFor(() => {
        expect(result.current.list.data).not.toContain(a.key);
      });
    });
  });

  describe("useDelete", () => {
    it("should correctly delete an agent", async () => {
      const a = await client.agents.create({ name: "toDelete" });

      const { result } = renderHook(() => Agent.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(a.key);
      });
      await waitFor(async () => {
        await expect(client.agents.retrieve(a.key)).rejects.toThrow(NotFoundError);
      });
    });

    it("should remove the agent from useRetrieve", async () => {
      const a = await client.agents.create({ name: "toDelete" });

      const { result } = renderHook(
        () => ({
          del: Agent.useDelete(),
          list: Agent.useList(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => expect(result.current.list.variant).toEqual("success"));
      expect(result.current.list.data).toContain(a.key);

      await act(async () => {
        await result.current.del.updateAsync(a.key);
      });

      await waitFor(() => {
        expect(result.current.list.data).not.toContain(a.key);
      });
    });
  });

  describe("useCreate", () => {
    it("should correctly create an agent", async () => {
      const { result } = renderHook(
        () => ({
          create: Agent.useCreate(),
          list: Agent.useList(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => expect(result.current.list.variant).toEqual("success"));
      const initialLength = result.current.list.data.length;

      await act(async () => {
        await result.current.create.updateAsync({ name: "newAgent" });
      });
      expect(result.current.create.variant).toEqual("success");

      await waitFor(() => {
        expect(result.current.list.data.length).toBeGreaterThan(initialLength);
      });
    });

    it("should make agent immediately available in list after creation", async () => {
      const { result } = renderHook(
        () => ({
          create: Agent.useCreate(),
          list: Agent.useList(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => expect(result.current.list.variant).toEqual("success"));

      await act(async () => {
        await result.current.create.updateAsync({ name: "immediateAgent" });
      });

      const allItems = result.current.list.data.map((k) =>
        result.current.list.getItem(k),
      );
      expect(allItems.some((a) => a?.name === "immediateAgent")).toBe(true);
    });
  });
});
