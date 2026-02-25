// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type label } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useCallback, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

describe("retrieve", () => {
  describe("useDirect", () => {
    describe("basic retrieval", () => {
      it("should return a loading result as its initial state", () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number>({
          name: "Resource",
          retrieve: async () => 0,
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), { wrapper });
        expect(result.current.variant).toEqual("loading");
        expect(result.current.data).toEqual(undefined);
        expect(result.current.status.message).toEqual("Retrieving Resource");
      });

      it("should return a success result when the data is fetched", async () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number>({
          name: "Resource",
          retrieve: async () => 12,
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), { wrapper });
        await waitFor(() => {
          expect(result.current.variant).toEqual("success");
          expect(result.current.data).toEqual(12);
          expect(result.current.status.message).toEqual(
            "Successfully retrieved Resource",
          );
        });
      });

      it("should return an error result when the retrieve function throws an error", async () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number>({
          name: "Resource",
          retrieve: async () => {
            throw new Error("test");
          },
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), { wrapper });
        await waitFor(() => {
          expect(result.current.variant).toEqual("error");
          expect(result.current.data).toEqual(undefined);
          expect(result.current.status.message).toEqual("Failed to retrieve Resource");
          expect(result.current.status.description).toEqual("test");
        });
      });

      it("should return an error result when no client is connected", async () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number>({
          name: "Resource",
          retrieve: async () => 0,
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), {
          wrapper: createSynnaxWrapper({ client: null }),
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("disabled");
          expect(result.current.data).toEqual(undefined);
          expect(result.current.status.message).toEqual("Failed to retrieve Resource");
          expect(result.current.status.description).toEqual(
            "Cannot retrieve Resource because no Core is connected.",
          );
        });
      });

      it("should allow null client when allowDisconnected is true", async () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number, {}, true>({
          name: "Resource",
          retrieve: async ({ client }) => {
            if (client == null) return 42;
            return 0;
          },
          allowDisconnected: true,
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), {
          wrapper: createSynnaxWrapper({ client: null }),
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("success");
          expect(result.current.data).toEqual(42);
        });
      });
    });

    interface Store extends Flux.Store {
      labels: Flux.UnaryStore<label.Key, label.Label>;
    }

    describe("listeners", () => {
      it("should correctly update the resource when the listener changes", async () => {
        const ch = await client.labels.create({
          name: "Test Label",
          color: color.construct("#000000"),
        });
        const { useRetrieve } = Flux.createRetrieve<
          { key: label.Key },
          label.Label,
          Store
        >({
          name: "Resource",
          retrieve: async ({ client, query: { key } }) =>
            await client.labels.retrieve({ key }),
          mountListeners: ({ store, onChange, query: { key } }) =>
            store.labels.onSet(onChange, key),
        });

        const { result } = renderHook(() => useRetrieve({ key: ch.key }), {
          wrapper,
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("success");
          expect(result.current.data).toEqual(ch);
        });
        await act(async () => {
          await client.labels.create({
            ...ch,
            name: "Test Label 2",
          });
        });
        await waitFor(
          () => {
            expect(result.current.data?.name).toEqual("Test Label 2");
            expect(
              result.current.variant,
              `${result.current.status.message}:${result.current.status.description}`,
            ).toEqual("success");
          },
          { timeout: 1000 },
        );
      });
    });
  });

  describe("useEffect", () => {
    it("should call the onChange handler when the data is fetched", async () => {
      const onChangeMock = vi.fn();
      const { result } = renderHook(
        () => {
          const [result, setResult] = useState<Flux.Result<number>>(
            Flux.loadingResult<number>("retrieving Resource", undefined),
          );
          const handleChange = useCallback(
            (value: Flux.Result<number>) => {
              setResult(value);
              onChangeMock(value);
            },
            [onChangeMock],
          );
          const { useRetrieveEffect } = Flux.createRetrieve<{ key: string }, number>({
            name: "Resource",
            retrieve: async () => 12,
          });
          useRetrieveEffect({
            query: { key: "test" },
            onChange: handleChange,
          });
          return result;
        },
        { wrapper },
      );
      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledTimes(2);
        expect(result.current.data).toEqual(12);
      });
    });
  });
});
