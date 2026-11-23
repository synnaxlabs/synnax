// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { calculation, channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, TimeSpan } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper, renderHookSuspended } from "@/testutil";

const path = Channel.formPath;
const calcPath = Channel.calculatedFormPath;

const client = createTestClient();
const other = createTestClient();

const createSource = async () =>
  await client.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    virtual: true,
  });

const createReset = async () =>
  await client.channels.create({
    name: id.create(),
    dataType: DataType.BOOLEAN,
    virtual: true,
  });

describe("channel forms", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await other.connect();
  });

  describe("useCalculatedForm", () => {
    it("should hold the zero values", () => {
      const { result } = renderHook(() => Channel.useCalculatedForm({ query: null }), {
        wrapper,
      });
      const v = result.current.form.value();
      expect(v.key).toBe(0);
      expect(v.name).toBe("");
      expect(v.expression).toBe("");
      expect(v.virtual).toBe(false);
      expect(v.isIndex).toBe(false);
      expect(v.index).toBe(0);
      expect(v.calculation).toBeUndefined();
      expect(v.operations).toEqual([
        { type: "none", resetChannel: 0, duration: TimeSpan.ZERO },
      ]);
    });

    it("should create a calculation with a windowed operation", async () => {
      const source = await createSource();
      const reset = await createReset();
      const { result } = renderHook(() => Channel.useCalculatedForm({ query: null }), {
        wrapper,
      });
      const name = id.create();
      act(() => {
        result.current.form.set(calcPath.name, name);
        result.current.form.set(calcPath.expression, `return ${source.name}`);
        result.current.form.set(calcPath.operations[0].type, "max");
        result.current.form.set(calcPath.operations[0].resetChannel, reset.key);
        result.current.form.set(calcPath.operations[0].duration, TimeSpan.seconds(3));
      });
      let saved = false;
      await act(async () => {
        saved = await result.current.saveAsync();
      });
      expect(saved).toBe(true);
      const key = result.current.form.value().key;
      expect(key).toBeGreaterThan(0);
      const [stored] = await client.calculations.retrieve({ output: key });
      expect(stored.operations).toEqual([
        { type: "max", reset: reset.key, duration: TimeSpan.seconds(3) },
      ]);
      expect(result.current.form.value().calculation).toEqual(stored.key);
      expect((await client.channels.retrieve(key)).name).toEqual(name);
    });

    it("should clear the operations when the form picks none", async () => {
      const source = await createSource();
      const calc = await client.calculations.create({
        name: id.create(),
        expression: `return ${source.name}`,
        operations: [{ type: "avg", duration: TimeSpan.seconds(2) }],
      });
      const { result } = await renderHookSuspended(
        () => Channel.useCalculatedForm({ query: { key: calc.output } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toBe("success"));
      expect(result.current.form.value().operations).toEqual([
        { type: "avg", resetChannel: 0, duration: TimeSpan.seconds(2) },
      ]);
      act(() => result.current.form.set(calcPath.operations[0].type, "none"));
      await act(async () => {
        await result.current.saveAsync();
      });
      await waitFor(async () => {
        const [stored] = await client.calculations.retrieve({ output: calc.output });
        expect(stored.operations).toEqual([]);
      });
    });

    it("should save a derivative without a window", async () => {
      const source = await createSource();
      const calc = await client.calculations.create({
        name: id.create(),
        expression: `return ${source.name}`,
      });
      const { result } = await renderHookSuspended(
        () => Channel.useCalculatedForm({ query: { key: calc.output } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toBe("success"));
      act(() => result.current.form.set(calcPath.operations[0].type, "derivative"));
      await act(async () => {
        await result.current.saveAsync();
      });
      await waitFor(async () => {
        const [stored] = await client.calculations.retrieve({ output: calc.output });
        expect(stored.operations).toEqual([{ type: "derivative" }]);
      });
    });

    it("should keep the second operation on a rename", async () => {
      const source = await createSource();
      const calc = await client.calculations.create({
        name: id.create(),
        expression: `return ${source.name}`,
        operations: [
          { type: "avg", duration: TimeSpan.seconds(2) },
          { type: "derivative" },
        ],
      });
      const { result } = await renderHookSuspended(
        () => Channel.useCalculatedForm({ query: { key: calc.output } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toBe("success"));
      const name = id.create();
      act(() => result.current.form.set(calcPath.name, name));
      await act(async () => {
        await result.current.saveAsync();
      });
      await waitFor(async () =>
        expect((await client.channels.retrieve(calc.output)).name).toEqual(name),
      );
      const [stored] = await client.calculations.retrieve({ output: calc.output });
      expect(stored.operations).toEqual([
        { type: "avg", duration: TimeSpan.seconds(2) },
        { type: "derivative" },
      ]);
    });

    it("should reload the operation and expression that another client sets", async () => {
      const source = await createSource();
      const calc = await client.calculations.create({
        name: id.create(),
        expression: `return ${source.name}`,
      });
      const { result } = await renderHookSuspended(
        () => Channel.useCalculatedForm({ query: { key: calc.output } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toBe("success"));
      await other.calculations.dispatch(calc.key, [
        calculation.set({ path: ["expression"], value: `return ${source.name} * 4` }),
        calculation.set({
          path: ["operations"],
          value: [{ type: "min", duration: TimeSpan.seconds(7) }],
        }),
      ]);
      await waitFor(() => {
        expect(result.current.form.value().expression).toEqual(
          `return ${source.name} * 4`,
        );
        expect(result.current.form.value().operations).toEqual([
          { type: "min", resetChannel: 0, duration: TimeSpan.seconds(7) },
        ]);
      });
      const name = id.create();
      await other.channels.dispatch(calc.output, channel.rename({ name }));
      await waitFor(() => expect(result.current.form.value().name).toEqual(name));
      expect(result.current.form.value().expression).toEqual(
        `return ${source.name} * 4`,
      );
      expect(result.current.form.value().calculation).toEqual(calc.key);
      // The reloaded values are the stored ones, so a save with no edit sends nothing.
      await act(async () => {
        await result.current.saveAsync();
      });
      const [stored] = await client.calculations.retrieve({ output: calc.output });
      expect(stored.expression).toEqual(`return ${source.name} * 4`);
      expect(stored.operations).toEqual([
        { type: "min", duration: TimeSpan.seconds(7) },
      ]);
    });

    it("should keep an unsaved expression when another client renames the channel", async () => {
      const source = await createSource();
      const calc = await client.calculations.create({
        name: id.create(),
        expression: `return ${source.name}`,
      });
      const { result } = await renderHookSuspended(
        () => Channel.useCalculatedForm({ query: { key: calc.output } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toBe("success"));
      act(() =>
        result.current.form.set(calcPath.expression, `return ${source.name} + 9`),
      );
      const name = id.create();
      await other.channels.dispatch(calc.output, channel.rename({ name }));
      await waitFor(() => expect(result.current.form.value().name).toEqual(name));
      expect(result.current.form.value().expression).toEqual(
        `return ${source.name} + 9`,
      );
      await act(async () => {
        await result.current.saveAsync();
      });
      const [stored] = await client.calculations.retrieve({ output: calc.output });
      expect(stored.expression).toEqual(`return ${source.name} + 9`);
      expect((await client.channels.retrieve(calc.output)).name).toEqual(name);
    });
  });

  describe("useForm", () => {
    it("should rename a calculated channel and leave its calculation", async () => {
      const source = await createSource();
      const calc = await client.calculations.create({
        name: id.create(),
        expression: `return ${source.name}`,
        operations: [{ type: "avg", duration: TimeSpan.seconds(2) }],
      });
      const { result } = await renderHookSuspended(
        () => Channel.useForm({ query: { key: calc.output } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toBe("success"));
      const name = id.create();
      act(() => result.current.form.set(path.name, name));
      let saved = false;
      await act(async () => {
        saved = await result.current.saveAsync();
      });
      expect(saved).toBe(true);
      await waitFor(async () =>
        expect((await client.channels.retrieve(calc.output)).name).toEqual(name),
      );
      const [stored] = await client.calculations.retrieve({ output: calc.output });
      expect(stored.expression).toEqual(`return ${source.name}`);
      expect(stored.operations).toEqual([
        { type: "avg", duration: TimeSpan.seconds(2) },
      ]);
    });

    it("should create a stored data channel with the zero values", async () => {
      const idx = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const { result } = renderHook(() => Channel.useForm({ query: null }), {
        wrapper,
      });
      const name = id.create();
      act(() => {
        result.current.form.set(path.name, name);
        result.current.form.set(path.index, idx.key);
      });
      let saved = false;
      await act(async () => {
        saved = await result.current.saveAsync();
      });
      expect(saved).toBe(true);
      const ch = await client.channels.retrieve(result.current.form.value().key);
      expect(ch.name).toEqual(name);
      expect(ch.virtual).toBe(false);
      expect(ch.internal).toBe(false);
      expect(ch.calculated).toBe(false);
      expect(ch.index).toEqual(idx.key);
      expect(ch.dataType.equals(DataType.FLOAT32)).toBe(true);
    });

    it("should reload a rename that another client makes, keeping an unsaved edit", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const { result } = await renderHookSuspended(
        () => Channel.useForm({ query: { key: ch.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toBe("success"));
      act(() => result.current.form.set(path.dataType, DataType.INT32.toString()));
      const name = id.create();
      await other.channels.dispatch(ch.key, channel.rename({ name }));
      await waitFor(() => expect(result.current.form.value().name).toEqual(name));
      expect(result.current.form.value().dataType).toEqual(DataType.INT32.toString());
    });
  });
});
