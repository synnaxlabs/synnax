// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DisconnectedError, newTestClient } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { newSynnaxWrapper } from "@/testutil/Synnax";

const client = newTestClient();

describe("retrieve", () => {
  describe("basic retrieval", () => {
    interface Params extends Flux.Params {}
    it("should return a loading result as its initial state", () => {
      const { result } = renderHook(
        () =>
          Flux.createRetrieve<Params, number>({
            name: "Resource",
            retrieve: async () => 0,
          }).useDirect({ params: {} }),
        { wrapper: newSynnaxWrapper(client) },
      );
      expect(result.current.variant).toEqual("loading");
      expect(result.current.data).toEqual(null);
      expect(result.current.error).toEqual(null);
      expect(result.current.message).toEqual("Retrieving Resource");
    });

    it("should return a success result when the data is fetched", async () => {
      const { result } = renderHook(
        () =>
          Flux.createRetrieve<Params, number>({
            name: "Resource",
            retrieve: async () => 12,
          }).useDirect({ params: {} }),
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual(12);
        expect(result.current.error).toEqual(null);
        expect(result.current.message).toEqual("Retrieved Resource");
      });
    });

    it("should return an error result when the retrieve function throws an error", async () => {
      const { result } = renderHook(
        () =>
          Flux.createRetrieve<Params, number>({
            name: "Resource",
            retrieve: async () => {
              throw new Error("test");
            },
          }).useDirect({ params: {} }),
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.error).toEqual(new Error("test"));
        expect(result.current.data).toEqual(null);
        expect(result.current.message).toEqual("Failed to retrieve Resource");
        expect(result.current.description).toEqual("test");
      });
    });

    it("should return an error result when no client is connected", async () => {
      const { result } = renderHook(
        () =>
          Flux.createRetrieve<Params, number>({
            name: "Resource",
            retrieve: async () => 0,
          }).useDirect({ params: {} }),
        { wrapper: newSynnaxWrapper(null) },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.error).toEqual(
          new DisconnectedError(
            "Cannot retrieve Resource because no cluster is connected.",
          ),
        );
        expect(result.current.data).toEqual(null);
        expect(result.current.message).toEqual("Failed to retrieve Resource");
        expect(result.current.description).toEqual(
          "Cannot retrieve Resource because no cluster is connected.",
        );
      });
    });
  });

  describe("listeners", () => {
    it("should correctly update the resource when the listener changes", async () => {
      const ch = await client.channels.create({
        name: "Test Channel",
        virtual: true,
        dataType: "float32",
      });
      interface Params extends Flux.Params {
        key: channel.Key;
      }
      const { result } = renderHook(
        () =>
          Flux.createRetrieve<Params, channel.Channel>({
            name: "Resource",
            retrieve: async ({ client, params: { key } }) =>
              await client.channels.retrieve(key),
            listeners: [
              {
                channel: channel.SET_CHANNEL_NAME,
                onChange: async ({ client, params: { key }, onChange }) => {
                  onChange(await client.channels.retrieve(key));
                },
              },
            ],
          }).useDirect({ params: { key: ch.key } }),
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual(ch);
      });
      await act(async () => {
        await client.channels.rename(ch.key, "Test Channel 2");
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data?.name).toEqual("Test Channel 2");
      });
    });

    it("should move the query into an error state when the listener throws an error", async () => {
      const ch = await client.channels.create({
        name: "Test Channel",
        virtual: true,
        dataType: "float32",
      });
      interface Params extends Flux.Params {
        key: channel.Key;
      }
      const { result } = renderHook(
        () =>
          Flux.createRetrieve<Params, channel.Channel>({
            name: "Resource",
            retrieve: async ({ client, params: { key } }) =>
              await client.channels.retrieve(key),
            listeners: [
              {
                channel: channel.SET_CHANNEL_NAME,
                onChange: async () => {
                  throw new Error("test");
                },
              },
            ],
          }).useDirect({ params: { key: ch.key } }),
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual(ch);
      });
      await act(async () => {
        await client.channels.rename(ch.key, "Test Channel 2");
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.error).toEqual(new Error("test"));
      });
    });
  });
});
