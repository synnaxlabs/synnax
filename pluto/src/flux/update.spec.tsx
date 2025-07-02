// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, newTestClient, type Synnax } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { Sync } from "@/flux/sync";
import { Synnax as PSynnax } from "@/synnax";

const client = newTestClient();

const newWrapper =
  (client: Synnax | null) =>
  // eslint-disable-next-line react/display-name
  (props: PropsWithChildren) => (
    <PSynnax.TestProvider client={client}>
      <Sync.Provider {...props} />
    </PSynnax.TestProvider>
  );

describe("update", () => {
  describe("basic update", () => {
    it("should return a success result as its initial state", () => {
      const { result } = renderHook(() =>
        Flux.createUpdate<{}, number>({
          name: "Resource",
          update: async () => {},
        }).useDirect({ params: {} }),
      );
      expect(result.current.variant).toEqual("success");
      expect(result.current.data).toEqual(null);
      expect(result.current.error).toEqual(null);
      expect(result.current.message).toEqual("Updated Resource");
    });

    it("should call update function when the user calls update", async () => {
      const update = vi.fn();
      const { result } = renderHook(
        () =>
          Flux.createUpdate<{}, number>({
            name: "Resource",
            update,
          }).useDirect({ params: {} }),
        { wrapper: newWrapper(client) },
      );
      act(() => result.current.update(12));
      await waitFor(() => {
        expect(update).toHaveBeenCalled();
        expect(result.current.data).toEqual(12);
      });
    });

    it("should return an error result if the update function throws an error", async () => {
      const update = vi.fn().mockRejectedValue(new Error("test"));
      const { result } = renderHook(
        () =>
          Flux.createUpdate<{}, number>({ name: "Resource", update }).useDirect({
            params: {},
          }),
        { wrapper: newWrapper(client) },
      );
      act(() => {
        result.current.update(12);
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.data).toEqual(null);
        expect(result.current.error).toEqual(new Error("test"));
        expect(result.current.message).toEqual("Failed to update Resource");
      });
    });

    it("should return an error result if the client is null and the update function is called", async () => {
      const update = vi.fn();
      const { result } = renderHook(
        () =>
          Flux.createUpdate<{}, number>({ name: "Resource", update }).useDirect({
            params: {},
          }),
        { wrapper: newWrapper(null) },
      );
      act(() => {
        result.current.update(12);
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.data).toEqual(null);
        expect(result.current.error).toEqual(
          new DisconnectedError(
            "Cannot update Resource because no cluster is connected.",
          ),
        );
        expect(result.current.message).toEqual("Failed to update Resource");
      });
    });

    it("should return a loading result when the update function is being executed", async () => {
      const update = async () => {
        await new Promise((resolve) => setTimeout(resolve, 10));
      };
      const { result } = renderHook(
        () =>
          Flux.createUpdate<{}, number>({ name: "Resource", update }).useDirect({
            params: {},
          }),
        { wrapper: newWrapper(client) },
      );
      act(() => {
        result.current.update(12);
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("loading");
        expect(result.current.data).toEqual(null);
        expect(result.current.error).toEqual(null);
        expect(result.current.message).toEqual("Updating Resource");
      });
    });
  });
});
