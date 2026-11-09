// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd, NotFoundError } from "@synnaxlabs/client";
import {
  createTestClient,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { ICD } from "@/icd";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createICD = async (name: string = `icd-${id.create()}`) =>
  await client.icds.create({
    name,
    messages: [{ name: "State", payload: { format: "binary" } }],
  });

describe("ICD queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("use", () => {
    it("should retrieve an ICD by key", async () => {
      const ic = await createICD();
      const { result } = await renderHookSuspended(() => ICD.use({ key: ic.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current?.key).toEqual(ic.key));
      expect(result.current?.messages).toEqual(ic.messages);
    });
  });

  describe("useList", () => {
    it("should list ICDs by key", async () => {
      const [a, b] = await Promise.all([createICD(), createICD()]);
      const { result } = renderHook(
        () => ICD.useList({ query: { where: { key: { in: [a.key, b.key] } } } }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() =>
        expect([...result.current.data].sort()).toEqual([a.key, b.key].sort()),
      );
    });

    it("should add an ICD created after the list loads", async () => {
      const key = uuid.create();
      const { result } = renderHook(
        () => ICD.useList({ query: { where: { key: { in: [key] } } } }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.answered).toBe(true));
      expect(result.current.data).toEqual([]);
      await client.icds.create({ key, name: `icd-${id.create()}` });
      await waitFor(() => expect(result.current.data).toEqual([key]));
    });
  });

  describe("useCreate", () => {
    it("should create an ICD", async () => {
      const key = uuid.create();
      const name = `icd-${id.create()}`;
      const { result } = renderHook(() => ICD.useCreate(), { wrapper });
      await act(async () => {
        expect(await result.current.updateAsync({ key, name })).toBe(true);
      });
      expect((await client.icds.retrieve({ key })).name).toEqual(name);
    });
  });

  describe("useRename", () => {
    it("should rename an ICD", async () => {
      const ic = await createICD();
      const { result } = await renderHookSuspended(
        () => ({
          name: ICD.useName({ key: ic.key }),
          rename: ICD.useRename(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.rename.updateAsync({ key: ic.key, name: "Renamed" });
      });
      await waitFor(() => expect(result.current.name).toEqual("Renamed"));
      expect((await client.icds.retrieve({ key: ic.key })).name).toEqual("Renamed");
    });
  });

  describe("useDelete", () => {
    it("should delete an ICD", async () => {
      const ic = await createICD();
      const { result } = renderHook(() => ICD.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(ic.key);
      });
      await expect(client.icds.retrieve({ key: ic.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("useImport", () => {
    it("should replace the messages of an ICD with an import", async () => {
      const ic = await createICD();
      const csv = "message,field,start_bit,bit_length\nEngine,Rpm,0,16";
      const { result } = renderHook(() => ICD.useImport(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({
          key: ic.key,
          format: "csv",
          data: new TextEncoder().encode(csv),
        });
      });
      const retrieved = await client.icds.retrieve({ key: ic.key });
      expect(retrieved.messages.map((m) => m.name)).toEqual(["Engine"]);
    });
  });

  describe("useForm", () => {
    beforeEach(async () => await waitForStreamLive(client.connection));

    const renderForm = async (key: icd.Key) =>
      await renderHookSuspended(() => ICD.useForm({ query: { key } }), { wrapper });

    it("should show the cached ICD", async () => {
      const ic = await createICD();
      const { result } = await renderForm(ic.key);
      expect(result.current.form.value()).toEqual({
        name: ic.name,
        messages: ic.messages,
      });
    });

    it("should send an edit to a message as a path action", async () => {
      const ic = await createICD();
      const [state] = ic.messages;
      const { result } = await renderForm(ic.key);
      const send = spyOnSend(client);
      act(() => result.current.form.set(ICD.formPath.messages[state.key].name, "Mode"));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(send.mock.calls.map(([target]) => target)).toContain("/icd/dispatch");
      const stored = await createTestClient().icds.retrieve({ key: ic.key });
      expect(stored.messages[0]).toEqual({ ...state, name: "Mode" });
    });

    it("should insert a message", async () => {
      const ic = await createICD();
      const { result } = await renderForm(ic.key);
      const status: icd.Message = {
        key: uuid.create(),
        name: "Status",
        payload: { format: "text", delimiter: ",", prefix: "$", fields: [] },
      };
      act(() =>
        result.current.form.insert(ICD.formPath.messages, status, ic.messages[0].key),
      );
      await act(async () => expect(await result.current.flush()).toBe(true));
      const stored = await createTestClient().icds.retrieve({ key: ic.key });
      expect(stored.messages.map(({ name }) => name)).toEqual(["State", "Status"]);
    });

    it("should store an edit that leaves a problem", async () => {
      const ic = await createICD();
      const { result } = await renderForm(ic.key);
      act(() =>
        result.current.form.insert(
          ICD.formPath.messages,
          { ...ic.messages[0], key: uuid.create() },
          ic.messages[0].key,
        ),
      );
      await act(async () => expect(await result.current.flush()).toBe(true));
      const stored = await createTestClient().icds.retrieve({ key: ic.key });
      expect(stored.messages.map(({ name }) => name)).toEqual(["State", "State"]);
    });

    it("should hold an empty name without sending it", async () => {
      const ic = await createICD();
      const { result } = await renderForm(ic.key);
      act(() => result.current.form.set(ICD.formPath.name, ""));
      await act(async () => expect(await result.current.flush()).toBe(false));
      expect((await createTestClient().icds.retrieve({ key: ic.key })).name).toEqual(
        ic.name,
      );
    });

    it("should show a rename made elsewhere", async () => {
      const ic = await createICD();
      const { result } = await renderForm(ic.key);
      await act(async () => await createTestClient().icds.rename(ic.key, "Renamed"));
      await waitFor(() => expect(result.current.form.value().name).toEqual("Renamed"));
    });

    it("should show an import made elsewhere", async () => {
      const ic = await createICD();
      const { result } = await renderForm(ic.key);
      const csv = "message,field,start_bit,bit_length\nEngine,Rpm,0,16";
      await act(
        async () =>
          await createTestClient().icds.import(
            ic.key,
            new TextEncoder().encode(csv),
            "csv",
          ),
      );
      await waitFor(() =>
        expect(result.current.form.value().messages.map(({ name }) => name)).toEqual([
          "Engine",
        ]),
      );
    });
  });
});
