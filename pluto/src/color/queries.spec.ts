// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Color } from "@/color";
import { Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const remote = createTestClient();

const createNamed = async (): Promise<color.Named> =>
  await remote.colors.create({ name: `color-${id.create()}`, light: "#00FF00" });

const renameRemotely = async (key: color.Key, name: string): Promise<void> => {
  await remote.colors.retrieve(key);
  await remote.colors.dispatch(key, color.set({ path: ["name"], value: name }));
};

describe("queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    await remote.connect();
  });

  describe("useRetrieve", () => {
    it("should show a rename from another client", async () => {
      const named = await createNamed();
      const { result } = await renderHookSuspended(
        () => Color.useRetrieve({ key: named.key }),
        { wrapper },
      );
      expect(result.current).toEqual(named);
      await act(async () => await renameRemotely(named.key, "Anomaly"));
      await waitFor(() => expect(result.current.name).toBe("Anomaly"));
    });
  });

  describe("useRetrieveMultiple", () => {
    it("should show a rename from another client", async () => {
      const [a, b] = await Promise.all([createNamed(), createNamed()]);
      const { result } = await renderHookSuspended(
        () => Color.useRetrieveMultiple({ keys: [a.key, b.key] }),
        { wrapper },
      );
      expect(result.current).toEqual([a, b]);
      await act(async () => await renameRemotely(a.key, "Anomaly"));
      await waitFor(() =>
        expect(result.current.map((n) => n.name)).toEqual(["Anomaly", b.name]),
      );
    });
  });

  describe("list", () => {
    it("should show a rename from another client", async () => {
      const named = await createNamed();
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.colors, {
            name: "named colors",
            query: { where: { key: { in: [named.key] } } },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([named.key]));
      await act(async () => await renameRemotely(named.key, "Anomaly"));
      await waitFor(() =>
        expect(result.current.getItem(named.key)?.name).toBe("Anomaly"),
      );
    });
  });

  describe("useCreate", () => {
    it("should create a named color", async () => {
      const { result } = renderHook(Color.useCreate, { wrapper });
      const name = `created-${id.create()}`;
      await act(async () => {
        await result.current.updateAsync({ name, light: "#FF0000" });
      });
      expect(result.current.variant).toEqual("success");
      expect((await createTestClient().colors.retrieve(name)).name).toBe(name);
    });
  });

  describe("useRename", () => {
    it("should rename a named color", async () => {
      const named = await createNamed();
      const { result } = renderHook(Color.useRename, { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: named.key, name: "Anomaly" });
      });
      expect(result.current.variant).toEqual("success");
      const stored = await createTestClient().colors.retrieve(named.key);
      expect(stored.name).toBe("Anomaly");
    });

    it("should fail for an empty name", async () => {
      const named = await createNamed();
      const { result } = renderHook(Color.useRename, { wrapper });
      await act(async () => {
        expect(await result.current.updateAsync({ key: named.key, name: "" })).toBe(
          false,
        );
      });
      expect(result.current.variant).toEqual("error");
    });
  });

  describe("useDelete", () => {
    it("should delete a named color", async () => {
      const named = await createNamed();
      const { result } = renderHook(Color.useDelete, { wrapper });
      await act(async () => {
        await result.current.updateAsync(named.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(createTestClient().colors.retrieve(named.key)).rejects.toThrow();
    });
  });
});
