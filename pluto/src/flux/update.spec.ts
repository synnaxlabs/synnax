// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cache } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type z } from "zod";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

const BASE_UPDATE_PARAMS: Pick<Flux.CreateUpdateParams<number>, "name" | "verbs"> = {
  name: "Resource",
  verbs: cache.UPDATE_VERBS,
};

describe("update", () => {
  let controller: AbortController;
  beforeEach(() => {
    controller = new AbortController();
  });
  afterEach(() => {
    controller.abort();
  });
  describe("updateSync", () => {
    it("should return a success result as its initial state", () => {
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update: async () => 0,
      });
      const { result } = renderHook(useUpdate, { wrapper });
      expect(result.current.variant).toEqual("success");
      expect(result.current.data).toEqual(undefined);
      expect(result.current.status.message).toEqual("Successfully updated Resource");
    });

    it("should call update function when the user calls update", async () => {
      const update = vi.fn();
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(useUpdate, { wrapper });
      act(() => result.current.update(12, { signal: controller.signal }));
      await waitFor(() => {
        expect(update).toHaveBeenCalled();
        expect(result.current.data).toEqual(12);
      });
    });

    it("should return an error result if the update function throws an error", async () => {
      const update = vi.fn().mockRejectedValue(new Error("test"));
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(useUpdate, { wrapper });
      act(() => {
        result.current.update(12, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.data).toEqual(undefined);
        expect(result.current.status.message).toEqual("Failed to update Resource");
      });
    });

    it("should return an error result if the client is null and the update function is called", async () => {
      const update = vi.fn();
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(useUpdate, {
        wrapper: createSynnaxWrapper({ client: null }),
      });
      act(() => {
        result.current.update(12, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("disabled");
        expect(result.current.data).toEqual(undefined);
        expect(result.current.status.message).toEqual("Failed to update Resource");
      });
    });

    it("should allow null client when allowDisconnected is true", async () => {
      const update = vi.fn().mockImplementation(async ({ client }) => {
        if (client == null) return 42;
        return 0;
      });
      const { useUpdate } = Flux.createUpdate<number, number, z.ZodNever, true>({
        ...BASE_UPDATE_PARAMS,
        update,
        allowDisconnected: true,
      });
      const { result } = renderHook(useUpdate, {
        wrapper: createSynnaxWrapper({ client: null }),
      });
      await act(async () => {
        await result.current.updateAsync(12, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual(12);
        expect(update).toHaveBeenCalled();
      });
    });

    it("should return a loading result when the update function is being executed", async () => {
      vi.useFakeTimers();
      try {
        let resolveUpdate!: () => void;
        const update = async () => {
          await new Promise<void>((resolve) => {
            resolveUpdate = resolve;
          });
          return 0;
        };
        const { useUpdate } = Flux.createUpdate<number>({
          ...BASE_UPDATE_PARAMS,
          update,
        });
        const { result } = renderHook(useUpdate, { wrapper });
        act(() => {
          result.current.update(12, { signal: controller.signal });
        });
        expect(result.current.data).toEqual(undefined);
        expect(result.current.variant).toEqual("loading");
        expect(result.current.status.message).toEqual("Updating Resource");
        act(() => resolveUpdate());
      } finally {
        vi.useRealTimers();
      }
    });
  });

  describe("updateAsync", () => {
    it("should return true if the update function is successful", async () => {
      const update = vi.fn();
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(useUpdate, { wrapper });
      const updated = await act(
        async () =>
          await result.current.updateAsync(12, {
            signal: controller.signal,
          }),
      );
      expect(updated).toEqual(true);
    });

    it("should return false if an error is thrown", async () => {
      const update = vi.fn().mockRejectedValue(new Error("test"));
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(useUpdate, { wrapper });
      const updated = await act(
        async () =>
          await result.current.updateAsync(12, {
            signal: controller.signal,
          }),
      );
      expect(updated).toEqual(false);
    });

    it("should return false if the client is null", async () => {
      const update = vi.fn();
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(useUpdate, {
        wrapper: createSynnaxWrapper({ client: null }),
      });
      const updated = await act(
        async () =>
          await result.current.updateAsync(12, {
            signal: controller.signal,
          }),
      );
      expect(updated).toEqual(false);
    });

    it("should return false if the update function is aborted", async () => {
      const update = vi.fn();
      const controller = new AbortController();
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(useUpdate, { wrapper });
      const updated = await act(async () => {
        controller.abort();
        return await result.current.updateAsync(12, {
          signal: controller.signal,
        });
      });
      expect(updated).toEqual(false);
    });
  });
  describe("rollback", () => {
    it("should execute rollbacks when update throws error", async () => {
      const rollback1 = vi.fn();
      const rollback2 = vi.fn();
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(rollback1);
        rollbacks.push(rollback2);
        return true;
      });
      const update = vi.fn().mockRejectedValue(new Error("update failed"));
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ beforeUpdate }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(rollback1).toHaveBeenCalled();
        expect(rollback2).toHaveBeenCalled();
      });
    });

    it("should execute rollbacks in reverse order (LIFO)", async () => {
      const order: number[] = [];
      const rollback1 = vi.fn(() => order.push(1));
      const rollback2 = vi.fn(() => order.push(2));
      const rollback3 = vi.fn(() => order.push(3));
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(rollback1);
        rollbacks.push(rollback2);
        rollbacks.push(rollback3);
        return true;
      });
      const update = vi.fn().mockRejectedValue(new Error("update failed"));
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ beforeUpdate }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(order).toEqual([3, 2, 1]);
    });

    it("should execute rollbacks when beforeUpdate returns false", async () => {
      const rollback = vi.fn();
      const update = vi.fn();
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(rollback);
        return false;
      });
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ beforeUpdate }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(rollback).toHaveBeenCalled();
      expect(update).not.toHaveBeenCalled();
    });
    it("should execute rollbacks when update returns false", async () => {
      const rollback = vi.fn();
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(rollback);
        return true;
      });
      const update = vi.fn().mockResolvedValue(false);
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ beforeUpdate }), { wrapper });
      const updated = await act(
        async () => await result.current.updateAsync(42, { signal: controller.signal }),
      );
      expect(updated).toBe(false);
      expect(rollback).not.toHaveBeenCalled();
    });
    it("should not execute rollbacks on successful update", async () => {
      const rollback = vi.fn();
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(rollback);
        return true;
      });
      const update = vi.fn().mockResolvedValue(42);
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ beforeUpdate }), { wrapper });
      const updated = await act(
        async () => await result.current.updateAsync(42, { signal: controller.signal }),
      );
      expect(updated).toBe(true);
      expect(rollback).not.toHaveBeenCalled();
    });
    it("should continue executing rollbacks even if one throws", async () => {
      const rollback1 = vi.fn(() => {
        throw new Error("rollback1 failed");
      });
      const rollback2 = vi.fn();
      const rollback3 = vi.fn();
      const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(rollback1);
        rollbacks.push(rollback2);
        rollbacks.push(rollback3);
        return true;
      });
      const update = vi.fn().mockRejectedValue(new Error("update failed"));
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ beforeUpdate }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(rollback1).toHaveBeenCalled();
      expect(rollback2).toHaveBeenCalled();
      expect(rollback3).toHaveBeenCalled();
      expect(consoleError).toHaveBeenCalledWith(
        "failed to rollback changes to Resource",
        expect.any(Error),
      );
      consoleError.mockRestore();
    });
    it("should not execute rollbacks when aborted via signal", async () => {
      const rollback = vi.fn();
      const abortController = new AbortController();
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(rollback);
        return true;
      });
      const update = vi.fn().mockImplementation(async () => {
        abortController.abort();
        return false;
      });
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ beforeUpdate }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: abortController.signal });
      });
      expect(rollback).not.toHaveBeenCalled();
    });

    it("should handle store mutations in rollbacks", async () => {
      const store: { value?: number } = {};
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        const oldValue = store.value;
        store.value = 100;
        rollbacks.push(() => {
          store.value = oldValue;
        });
        return true;
      });
      const update = vi.fn().mockRejectedValue(new Error("update failed"));
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ beforeUpdate }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(store.value).toBeUndefined();
    });

    it("should execute rollbacks from beforeUpdate and afterOptimistic when update throws", async () => {
      const beforeRollback = vi.fn();
      const optimisticRollback = vi.fn();
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(beforeRollback);
        return true;
      });
      const afterOptimistic = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(optimisticRollback);
      });
      const update = vi.fn().mockImplementation(async ({ onOptimisticComplete }) => {
        await onOptimisticComplete(42);
        throw new Error("update failed");
      });
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(
        () => useUpdate({ beforeUpdate, afterOptimistic }),
        {
          wrapper,
        },
      );
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(beforeRollback).toHaveBeenCalled();
      expect(optimisticRollback).toHaveBeenCalled();
    });

    it("should pass modified data from beforeUpdate to update", async () => {
      const update = vi.fn().mockResolvedValue(100);
      const beforeUpdate = vi.fn().mockResolvedValue(99);
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ beforeUpdate }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(beforeUpdate).toHaveBeenCalledWith(expect.objectContaining({ data: 42 }));
      expect(update).toHaveBeenCalledWith(expect.objectContaining({ data: 99 }));
    });
  });

  describe("afterOptimistic", () => {
    it("should call afterOptimistic when the update invokes onOptimisticComplete", async () => {
      const afterOptimistic = vi.fn();
      const update = vi.fn().mockImplementation(async ({ onOptimisticComplete }) => {
        await onOptimisticComplete(42);
        return 42;
      });
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ afterOptimistic }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(afterOptimistic).toHaveBeenCalledOnce();
    });

    it("should not call afterOptimistic when the update never invokes onOptimisticComplete", async () => {
      const afterOptimistic = vi.fn();
      const update = vi.fn().mockResolvedValue(42);
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ afterOptimistic }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(afterOptimistic).not.toHaveBeenCalled();
    });

    it("should call afterOptimistic before afterSuccess", async () => {
      const order: string[] = [];
      const afterOptimistic = vi.fn(() => {
        order.push("optimistic");
      });
      const afterSuccess = vi.fn(() => {
        order.push("success");
      });
      const update = vi.fn().mockImplementation(async ({ onOptimisticComplete }) => {
        await onOptimisticComplete(42);
        return 42;
      });
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(
        () => useUpdate({ afterOptimistic, afterSuccess }),
        { wrapper },
      );
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(order).toEqual(["optimistic", "success"]);
    });

    it("should pass the optimistic output, client, and rollbacks to afterOptimistic", async () => {
      const afterOptimistic = vi.fn();
      const update = vi.fn().mockImplementation(async ({ onOptimisticComplete }) => {
        await onOptimisticComplete(99);
        return 99;
      });
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(() => useUpdate({ afterOptimistic }), { wrapper });
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      expect(afterOptimistic).toHaveBeenCalledWith(
        expect.objectContaining({
          client,
          data: 99,
          rollbacks: expect.any(Array),
        }),
      );
    });

    it("should run rollbacks and surface an error when afterOptimistic throws", async () => {
      const rollback = vi.fn();
      const beforeUpdate = vi.fn().mockImplementation(({ rollbacks }) => {
        rollbacks.push(rollback);
        return true;
      });
      const afterOptimistic = vi.fn().mockRejectedValue(new Error("optimistic failed"));
      const update = vi.fn().mockImplementation(async ({ onOptimisticComplete }) => {
        await onOptimisticComplete(42);
        return 42;
      });
      const { useUpdate } = Flux.createUpdate<number>({
        ...BASE_UPDATE_PARAMS,
        update,
      });
      const { result } = renderHook(
        () => useUpdate({ beforeUpdate, afterOptimistic }),
        {
          wrapper,
        },
      );
      await act(async () => {
        await result.current.updateAsync(42, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(rollback).toHaveBeenCalled();
        expect(result.current.variant).toEqual("error");
      });
    });
  });
});
