// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { calculation, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, sleep, TimeSpan } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Channel } from "@/channel";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = Channel.calculatedFormPath;

describe("useCalculatedForm breaker", () => {
  const client = createTestClient();
  const other = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createSource = async () =>
    await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });

  const openForm = async (key: number) => {
    const rendered = await renderHookSuspended(
      () => Channel.useCalculatedForm({ query: { key } }),
      { wrapper },
    );
    await waitFor(() => expect(rendered.result.current.variant).toEqual("success"));
    return rendered;
  };

  it("should reload the form when the calculation's expression changes elsewhere", async () => {
    const source = await createSource();
    const calc = await client.calculations.create({
      name: id.create(),
      expression: `return ${source.name} * 2`,
    });
    const { result } = await openForm(calc.output);
    expect(result.current.form.value().expression).toEqual(`return ${source.name} * 2`);
    await other.calculations.retrieve({ key: calc.key });
    await other.calculations.dispatch(
      calc.key,
      calculation.set({ path: ["expression"], value: `return ${source.name} * 3` }),
    );
    // The change reaches the client's calculation cache.
    await waitFor(() =>
      expect(client.calculations.watch({ key: calc.key }).getCached()).toMatchObject({
        expression: `return ${source.name} * 3`,
      }),
    );
    await waitFor(
      () =>
        expect(result.current.form.value().expression).toEqual(
          `return ${source.name} * 3`,
        ),
      { timeout: 5000 },
    );
  });

  it("should keep the calculation's expression and operations across a rename", async () => {
    const source = await createSource();
    const calc = await client.calculations.create({
      name: id.create(),
      expression: `return ${source.name} + 1`,
      operations: [{ type: "avg", duration: TimeSpan.seconds(5) }],
    });
    // A tree or a plot holds the channel in the cache before the form opens.
    await client.channels.retrieve(calc.output);
    await client.calculations.dispatch(
      calc.key,
      calculation.set({ path: ["expression"], value: `return ${source.name} + 2` }),
    );
    const { result } = await openForm(calc.output);
    expect(result.current.form.value().expression).toEqual(`return ${source.name} + 2`);
    expect(result.current.form.value().operations[0].type).toEqual("avg");
    const name = id.create();
    await act(async () => await client.channels.rename(calc.output, name));
    await waitFor(() => expect(result.current.form.value().name).toEqual(name));
    expect(result.current.form.value().expression).toEqual(`return ${source.name} + 2`);
    expect(result.current.form.value().operations[0].type).toEqual("avg");
  });

  it("should save the removal of an operation typed before a rename elsewhere", async () => {
    const source = await createSource();
    const calc = await client.calculations.create({
      name: id.create(),
      expression: `return ${source.name} + 1`,
      operations: [{ type: "avg", duration: TimeSpan.seconds(5) }],
    });
    const { result } = await openForm(calc.output);
    expect(result.current.form.value().operations[0].type).toEqual("avg");
    act(() => {
      result.current.form.set(path.expression, `return ${source.name} + 3`);
      result.current.form.set(path.operations[0].type, "none");
    });
    const name = id.create();
    await act(async () => await client.channels.rename(calc.output, name));
    await waitFor(() => expect(result.current.form.value().name).toEqual(name));
    let saved = false;
    await act(async () => {
      saved = await result.current.saveAsync();
    });
    expect(saved).toBe(true);
    const [stored] = await client.calculations.retrieve({ output: calc.output });
    expect(stored.expression).toEqual(`return ${source.name} + 3`);
    expect(stored.operations).toEqual([]);
  });

  it("should keep the calculation's expression after the form saves a rename", async () => {
    const source = await createSource();
    const calc = await client.calculations.create({
      name: id.create(),
      expression: `return ${source.name} + 1`,
    });
    await client.channels.retrieve(calc.output);
    await client.calculations.dispatch(
      calc.key,
      calculation.set({ path: ["expression"], value: `return ${source.name} + 2` }),
    );
    const { result } = await openForm(calc.output);
    act(() => result.current.form.set(path.name, id.create()));
    await act(async () => {
      expect(await result.current.saveAsync()).toBe(true);
    });
    // The rename reaches the cache through the stream after the save resolves.
    await sleep.sleep(TimeSpan.milliseconds(500));
    expect(result.current.form.value().expression).toEqual(`return ${source.name} + 2`);
  });

  it("should not report a save as done when the calculation was deleted while open", async () => {
    const source = await createSource();
    const calc = await client.calculations.create({
      name: id.create(),
      expression: `return ${source.name} * 2`,
    });
    const { result } = await openForm(calc.output);
    await client.calculations.delete(calc.key);
    act(() => result.current.form.set(path.expression, `return ${source.name} * 4`));
    let saved = false;
    await act(async () => {
      saved = await result.current.saveAsync();
    });
    const stored = await client.calculations.retrieve({ output: calc.output });
    // A save that reports success holds the expression the user saved.
    if (saved)
      expect(stored.map((c) => c.expression)).toContain(`return ${source.name} * 4`);
  });
});
