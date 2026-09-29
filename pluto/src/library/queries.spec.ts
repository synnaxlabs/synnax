// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, library, NotFoundError, ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Library } from "@/library";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createLibrary = async (name: string = `library-${id.create()}`) =>
  await client.libraries.create({
    name,
    entries: [
      {
        kind: "enum",
        name: "State",
        values: [
          { value: 0, name: "Off" },
          { value: 1, name: "On" },
        ],
      },
    ],
  });

describe("Library queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("use", () => {
    it("should retrieve a library by key", async () => {
      const lib = await createLibrary();
      const { result } = await renderHookSuspended(
        () => Library.use({ key: lib.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current?.key).toEqual(lib.key));
      expect(result.current?.entries).toEqual(lib.entries);
    });
  });

  describe("useList", () => {
    it("should list libraries by key", async () => {
      const [a, b] = await Promise.all([createLibrary(), createLibrary()]);
      const { result } = renderHook(() => Library.useList(), { wrapper });
      act(() => result.current.retrieve({ keys: [a.key, b.key] }));
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect([...result.current.data].sort()).toEqual([a.key, b.key].sort());
    });

    it("should add a library created after the list loads", async () => {
      const { result } = renderHook(() => Library.useList(), { wrapper });
      act(() => result.current.retrieve({}));
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const lib = await createLibrary();
      await waitFor(() => expect(result.current.data).toContain(lib.key));
    });
  });

  describe("useCreate", () => {
    it("should create a library", async () => {
      const key = uuid.create();
      const name = `library-${id.create()}`;
      const { result } = renderHook(() => Library.useCreate(), { wrapper });
      await act(async () => {
        expect(await result.current.updateAsync({ key, name })).toBe(true);
      });
      expect((await client.libraries.retrieve({ key })).name).toEqual(name);
    });
  });

  describe("useRename", () => {
    it("should rename a library", async () => {
      const lib = await createLibrary();
      const { result } = await renderHookSuspended(
        () => ({
          name: Library.useName({ key: lib.key }),
          rename: Library.useRename(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.rename.updateAsync({ key: lib.key, name: "Renamed" });
      });
      await waitFor(() => expect(result.current.name).toEqual("Renamed"));
      expect((await client.libraries.retrieve({ key: lib.key })).name).toEqual(
        "Renamed",
      );
    });
  });

  describe("useDelete", () => {
    it("should delete a library", async () => {
      const lib = await createLibrary();
      const { result } = renderHook(() => Library.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(lib.key);
      });
      await expect(client.libraries.retrieve({ key: lib.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("useImport", () => {
    it("should replace the entries of a library with an import", async () => {
      const lib = await createLibrary();
      const csv = "message,field,start_bit,bit_length\nEngine,Rpm,0,16";
      const { result } = renderHook(() => Library.useImport(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({
          key: lib.key,
          format: "csv",
          data: new TextEncoder().encode(csv),
        });
      });
      const retrieved = await client.libraries.retrieve({ key: lib.key });
      expect(retrieved.entries.map((e) => e.name)).toEqual(["Engine"]);
    });
  });

  describe("useForm", () => {
    it("should save the whole library", async () => {
      const lib = await createLibrary();
      const { result } = await renderHookSuspended(
        () => Library.useForm({ query: { key: lib.key } }),
        { wrapper },
      );
      expect(result.current.form.value().entries).toEqual(lib.entries);
      act(() =>
        result.current.form.set("entries", [
          ...lib.entries,
          {
            kind: "message",
            key: uuid.create(),
            name: "Status",
            format: "binary",
            delimiter: ",",
            fields: [
              {
                encoding: "binary",
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
        ]),
      );
      await act(async () => {
        expect(await result.current.saveAsync()).toBe(true);
      });
      const retrieved = await client.libraries.retrieve({ key: lib.key });
      expect(retrieved.entries).toEqual(result.current.form.value().entries);
      expect(retrieved.entries.map((e) => e.name)).toEqual(["State", "Status"]);
    });

    it("should place a Core validation error on the field it names", async () => {
      const lib = await createLibrary();
      const [state] = lib.entries;
      const { result } = await renderHookSuspended(
        () => Library.useForm({ query: { key: lib.key } }),
        { wrapper },
      );
      act(() =>
        result.current.form.set("entries", [state, { ...state, key: uuid.create() }]),
      );
      await act(async () => {
        expect(await result.current.saveAsync()).toBe(false);
      });
      const second = result.current.form.value().entries[1].key;
      const status = result.current.form.get(`entries.${second}.name`).status;
      expect(status).toMatchObject({
        variant: "error",
        message: 'duplicate entry name "State": validation error',
      });
    });

    it("should take a rename made elsewhere", async () => {
      const lib = await createLibrary();
      const { result } = await renderHookSuspended(
        () => Library.useForm({ query: { key: lib.key } }),
        { wrapper },
      );
      await client.libraries.rename(lib.key, "Renamed");
      await waitFor(() => expect(result.current.form.value().name).toEqual("Renamed"));
    });
  });

  describe("useTasks", () => {
    it("should not list resources joined by other relationships", async () => {
      const lib = await createLibrary();
      const child = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `group-${id.create()}`,
      });
      await client.ontology.addChildren(
        library.ontologyID(lib.key),
        group.ontologyID(child.key),
      );
      const { result } = await renderHookSuspended(
        () => Library.useTasks({ key: lib.key }),
        { wrapper },
      );
      expect(result.current).toEqual([]);
    });
  });
});
