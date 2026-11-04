// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, StaleICDError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { assert, beforeEach, describe, expect, it } from "vitest";

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
      const { result } = renderHook(() => ICD.useList(), { wrapper });
      act(() => result.current.retrieve({ keys: [a.key, b.key] }));
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect([...result.current.data].sort()).toEqual([a.key, b.key].sort());
    });

    it("should add an ICD created after the list loads", async () => {
      const { result } = renderHook(() => ICD.useList(), { wrapper });
      act(() => result.current.retrieve({}));
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const ic = await createICD();
      await waitFor(() => expect(result.current.data).toContain(ic.key));
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
    it("should save the whole ICD", async () => {
      const ic = await createICD();
      const { result } = await renderHookSuspended(
        () => ICD.useForm({ query: { key: ic.key } }),
        { wrapper },
      );
      expect(result.current.form.value().messages).toEqual(ic.messages);
      act(() =>
        result.current.form.set(ICD.formPath.messages, [
          ...ic.messages,
          {
            key: uuid.create(),
            name: "Status",
            payload: {
              format: "binary",
              fields: [
                {
                  key: uuid.create(),
                  name: "state",
                  startBit: 0,
                  bitLength: 8,
                  byteOrder: "little_endian",
                  signed: false,
                  float: false,
                  scale: 1,
                  offset: 0,
                  units: "",
                  multiplexValues: [],
                },
              ],
            },
          },
        ]),
      );
      await act(async () => {
        expect(await result.current.saveAsync()).toBe(true);
      });
      const retrieved = await client.icds.retrieve({ key: ic.key });
      expect(retrieved.messages).toEqual(result.current.form.value().messages);
      expect(retrieved.messages.map((m) => m.name)).toEqual(["State", "Status"]);
    });

    it("should put each problem on the field it names and not save", async () => {
      const ic = await createICD();
      const [state] = ic.messages;
      const { result } = await renderHookSuspended(
        () => ICD.useForm({ query: { key: ic.key } }),
        { wrapper },
      );
      act(() =>
        result.current.form.set(ICD.formPath.messages, [
          state,
          { ...state, key: uuid.create() },
        ]),
      );
      await act(async () => {
        expect(await result.current.saveAsync()).toBe(false);
      });
      const second = result.current.form.value().messages[1].key;
      const status = result.current.form.get(ICD.formPath.messages[second].name).status;
      expect(status).toMatchObject({
        variant: "error",
        message: 'duplicate message name "State"',
      });
      expect(result.current.status.description).toEqual("The ICD has 1 problem");
      expect((await client.icds.retrieve({ key: ic.key })).messages).toEqual(
        ic.messages,
      );
    });

    it("should reject a save when the ICD changed elsewhere", async () => {
      const ic = await createICD();
      const { result } = await renderHookSuspended(
        () => ICD.useForm({ query: { key: ic.key } }),
        { wrapper },
      );
      await client.icds.create({ ...ic, messages: [] });
      act(() =>
        result.current.form.set(ICD.formPath.messages[ic.messages[0].key].name, "Mode"),
      );
      await act(async () => {
        expect(await result.current.saveAsync()).toBe(false);
      });
      const { status } = result.current;
      assert(status.variant === "error");
      expect(StaleICDError.matches(status.details.error)).toBe(true);
      expect((await client.icds.retrieve({ key: ic.key })).messages).toEqual([]);
    });

    it("should save after a rename made elsewhere", async () => {
      const ic = await createICD();
      const { result } = await renderHookSuspended(
        () => ICD.useForm({ query: { key: ic.key } }),
        { wrapper },
      );
      await client.icds.rename(ic.key, "Renamed");
      await waitFor(() => expect(result.current.form.value().name).toEqual("Renamed"));
      act(() =>
        result.current.form.set(ICD.formPath.messages[ic.messages[0].key].name, "Mode"),
      );
      await act(async () => {
        expect(await result.current.saveAsync()).toBe(true);
      });
      const retrieved = await client.icds.retrieve({ key: ic.key });
      expect(retrieved.name).toEqual("Renamed");
      expect(retrieved.messages[0].name).toEqual("Mode");
    });

    it("should take a rename made elsewhere", async () => {
      const ic = await createICD();
      const { result } = await renderHookSuspended(
        () => ICD.useForm({ query: { key: ic.key } }),
        { wrapper },
      );
      await client.icds.rename(ic.key, "Renamed");
      await waitFor(() => expect(result.current.form.value().name).toEqual("Renamed"));
    });
  });
});
