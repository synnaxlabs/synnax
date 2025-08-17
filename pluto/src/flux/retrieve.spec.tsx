// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, newTestClient } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { newSynnaxWrapper } from "@/testutil/Synnax";

const client = newTestClient();

describe("retrieve", () => {
  describe("useDirect", () => {
    describe("basic retrieval", () => {
      it("should return a loading result as its initial state", () => {
        const { result } = renderHook(
          () =>
            Flux.createRetrieve<{}, number>({
              name: "Resource",
              retrieve: async () => 0,
            }).useDirect({ params: {} }),
          { wrapper: newSynnaxWrapper(client) },
        );
        expect(result.current.variant).toEqual("loading");
        expect(result.current.data).toEqual(null);
        expect(result.current.status.message).toEqual("Retrieving Resource");
      });

      it("should return a success result when the data is fetched", async () => {
        const { result } = renderHook(
          () =>
            Flux.createRetrieve<{}, number>({
              name: "Resource",
              retrieve: async () => 12,
            }).useDirect({ params: {} }),
          { wrapper: newSynnaxWrapper(client) },
        );
        await waitFor(() => {
          expect(result.current.variant).toEqual("success");
          expect(result.current.data).toEqual(12);
          expect(result.current.status.message).toEqual("Retrieved Resource");
        });
      });

      it("should return an error result when the retrieve function throws an error", async () => {
        const { result } = renderHook(
          () =>
            Flux.createRetrieve<{}, number>({
              name: "Resource",
              retrieve: async () => {
                throw new Error("test");
              },
            }).useDirect({ params: {} }),
          { wrapper: newSynnaxWrapper(client) },
        );
        await waitFor(() => {
          expect(result.current.variant).toEqual("error");
          expect(result.current.data).toEqual(null);
          expect(result.current.status.message).toEqual("Failed to retrieve Resource");
          expect(result.current.status.description).toEqual("test");
        });
      });

      it("should return an error result when no client is connected", async () => {
        const { result } = renderHook(
          () =>
            Flux.createRetrieve<{}, number>({
              name: "Resource",
              retrieve: async () => 0,
            }).useDirect({ params: {} }),
          { wrapper: newSynnaxWrapper(null) },
        );
        await waitFor(() => {
          expect(result.current.variant).toEqual("error");
          expect(result.current.data).toEqual(null);
          expect(result.current.status.message).toEqual("Failed to retrieve Resource");
          expect(result.current.status.description).toEqual(
            "Cannot retrieve Resource because no cluster is connected.",
          );
        });
      });
    });

    interface Store extends Flux.Store {
      labels: Flux.UnaryStore<label.Key, label.Label>;
    }

    const SET_LABEL_LISTENER: Flux.ChannelListener<Store, typeof label.labelZ> = {
      channel: label.SET_CHANNEL_NAME,
      schema: label.labelZ,
      onChange: ({ store, changed }) => store.labels.set(changed.key, changed),
    };

    const storeConfig: Flux.StoreConfig<Store> = {
      labels: {
        listeners: [SET_LABEL_LISTENER],
      },
    };

    describe("listeners", () => {
      it("should correctly update the resource when the listener changes", async () => {
        const ch = await client.labels.create({
          name: "Test Label",
          color: "#000000",
        });
        const { result } = renderHook(
          () =>
            Flux.createRetrieve<{ key: label.Key }, label.Label, Store>({
              name: "Resource",
              retrieve: async ({ client, params: { key } }) =>
                await client.labels.retrieve({ key }),
              mountListeners: ({ store, onChange, params: { key } }) =>
                store.labels.onSet(onChange, key),
            }).useDirect({ params: { key: ch.key } }),
          { wrapper: newSynnaxWrapper(client, storeConfig) },
        );
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
            Flux.pendingResult<number>("Resource", "retrieving", null),
          );
          const handleChange: Flux.UseEffectRetrieveArgs<
            { key: string },
            number
          >["onChange"] = (value) => {
            setResult(value);
            onChangeMock(value);
          };

          Flux.createRetrieve<
            {
              key: string;
            },
            number
          >({
            name: "Resource",
            retrieve: async () => 12,
          }).useEffect({ params: { key: "test" }, onChange: handleChange });
          return result;
        },
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledTimes(2);
        expect(result.current.data).toEqual(12);
      });
    });
  });
});
