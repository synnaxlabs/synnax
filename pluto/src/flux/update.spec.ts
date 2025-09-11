// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

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
        name: "Resource",
        update: async () => {},
      });
      const { result } = renderHook(useUpdate, { wrapper });
      expect(result.current.variant).toEqual("success");
      expect(result.current.data).toEqual(undefined);
      expect(result.current.status.message).toEqual("Updated Resource");
    });

    it("should call update function when the user calls update", async () => {
      const update = vi.fn();
      const { useUpdate } = Flux.createUpdate<number>({ name: "Resource", update });
      const { result } = renderHook(useUpdate, { wrapper });
      act(() => result.current.update(12, { signal: controller.signal }));
      await waitFor(() => {
        expect(update).toHaveBeenCalled();
        expect(result.current.data).toEqual(12);
      });
    });

    it("should return an error result if the update function throws an error", async () => {
      const update = vi.fn().mockRejectedValue(new Error("test"));
      const { useUpdate } = Flux.createUpdate<number>({ name: "Resource", update });
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
      const { useUpdate } = Flux.createUpdate<number>({ name: "Resource", update });
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

    it("should return a loading result when the update function is being executed", async () => {
      const update = async () => {
        await new Promise((resolve) => setTimeout(resolve, 10));
      };
      const { useUpdate } = Flux.createUpdate<number>({ name: "Resource", update });
      const { result } = renderHook(useUpdate, { wrapper });
      act(() => {
        result.current.update(12, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.data).toEqual(undefined);
        expect(result.current.variant).toEqual("loading");
        expect(result.current.status.message).toEqual("Updating Resource");
      });
    });
  });

  describe("updateAsync", () => {
    it("should return true if the update function is successful", async () => {
      const update = vi.fn();
      const { useUpdate } = Flux.createUpdate<number>({ name: "Resource", update });
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
      const { useUpdate } = Flux.createUpdate<number>({ name: "Resource", update });
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
      const { useUpdate } = Flux.createUpdate<number>({ name: "Resource", update });
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
      const { useUpdate } = Flux.createUpdate<number>({ name: "Resource", update });
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
});
