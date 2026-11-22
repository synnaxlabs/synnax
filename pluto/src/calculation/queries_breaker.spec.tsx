// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { calculation, type channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { id, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useState } from "react";
import { beforeAll, describe, expect, it, onTestFinished, vi } from "vitest";

import { Calculation } from "@/calculation";
import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const LOOKUP = "/calculation/retrieve";

describe("useNonOutputFilter breaker", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createPlain = async () =>
    await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });

  // Wires the filter into a picker the way the Sift pickers do.
  const Picker = ({
    query,
    value = [],
  }: {
    query: Channel.ListQuery;
    value?: channel.Key[];
  }): ReactElement => {
    const [open, setOpen] = useState(false);
    const stored = Calculation.useNonOutputFilter({ open });
    return (
      <Channel.SelectMultiple
        value={value}
        onChange={() => {}}
        query={query}
        filter={stored.filter}
        status={stored.status}
        visible={open}
        onVisibleChange={setOpen}
        virtual={false}
      />
    );
  };

  it("should not send lookups in a loop while the Core fails them", async () => {
    const plain = await createPlain();
    const { unary } = client.transport;
    const send = unary.send.bind(unary);
    let lookups = 0;
    const spy = vi.spyOn(unary, "send").mockImplementation(async (...args) => {
      if (args[0] !== LOOKUP) return await send(...args);
      lookups++;
      throw new Error("calculations unavailable");
    });
    onTestFinished(() => spy.mockRestore());
    const c = render(<Picker query={{ where: { key: { in: [plain.key] } } }} />, {
      wrapper,
    });
    fireEvent.click(c.getAllByRole("button")[0]);
    await waitFor(() => expect(lookups).toBeGreaterThan(0));
    // The list does not change, so no new page starts a new batch.
    await new Promise((resolve) => setTimeout(resolve, 1000));
    expect(lookups).toBeLessThanOrEqual(2);
  });

  it("should keep showing a failed lookup when a channel joins the selection", async () => {
    const failing = await createPlain();
    const picked = await createPlain();
    // The picked channel has its answer before the lookups fail.
    await client.channels.retrieve(picked.key);
    await client.calculations.outputs.retrieve([picked.key]);
    const { unary } = client.transport;
    const send = unary.send.bind(unary);
    const spy = vi.spyOn(unary, "send").mockImplementation(async (...args) => {
      if (args[0] === LOOKUP) throw new Error("calculations unavailable");
      return await send(...args);
    });
    onTestFinished(() => spy.mockRestore());
    const query = { where: { key: { in: [failing.key] } } };
    const c = render(<Picker query={query} />, { wrapper });
    fireEvent.click(c.getAllByRole("button")[0]);
    expect(await c.findByText("Failed to load channels")).toBeTruthy();
    // A channel dropped on the trigger joins the selection; the list is unchanged.
    c.rerender(<Picker query={query} value={[picked.key]} />);
    expect(await c.findByText(picked.name)).toBeTruthy();
    await new Promise((resolve) => setTimeout(resolve, 200));
    expect(c.queryByText(/No channels/)).toBeNull();
    expect(c.queryByText("Failed to load channels")).not.toBeNull();
  });

  it("should keep showing a failed lookup when a selected channel is renamed", async () => {
    const term = id.create();
    const failing = await client.channels.create({
      name: `${term}_failing`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const picked = await createPlain();
    await client.channels.retrieve(picked.key);
    await client.calculations.outputs.retrieve([picked.key]);
    const { unary } = client.transport;
    const send = unary.send.bind(unary);
    const spy = vi.spyOn(unary, "send").mockImplementation(async (...args) => {
      if (args[0] === LOOKUP) throw new Error("calculations unavailable");
      return await send(...args);
    });
    onTestFinished(() => spy.mockRestore());
    const query = { where: { key: { in: [failing.key] } } };
    const c = render(<Picker query={query} value={[picked.key]} />, { wrapper });
    fireEvent.click(c.getAllByRole("button")[0]);
    // A search lists only the matches, so the selected channel leaves the list.
    fireEvent.change(await c.findByPlaceholderText(/search/i), {
      target: { value: term },
    });
    expect(await c.findByText("Failed to load channels")).toBeTruthy();
    const renamed = id.create();
    await client.channels.rename(picked.key, renamed);
    expect(await c.findByText(renamed)).toBeTruthy();
    await new Promise((resolve) => setTimeout(resolve, 200));
    expect(c.queryByText(/No channels/)).toBeNull();
    expect(c.queryByText("Failed to load channels")).not.toBeNull();
  });

  it("should look a failed channel up again when the picker opens again", async () => {
    const plain = await createPlain();
    const { unary } = client.transport;
    const send = unary.send.bind(unary);
    const spy = vi.spyOn(unary, "send").mockImplementation(async (...args) => {
      if (args[0] === LOOKUP) throw new Error("calculations unavailable");
      return await send(...args);
    });
    onTestFinished(() => spy.mockRestore());
    const c = render(<Picker query={{ where: { key: { in: [plain.key] } } }} />, {
      wrapper,
    });
    const trigger = c.getAllByRole("button")[0];
    fireEvent.click(trigger);
    expect(await c.findByText("Failed to load channels")).toBeTruthy();
    // The Core answers again, and the person follows the description: they open the
    // picker again.
    spy.mockRestore();
    fireEvent.click(trigger);
    await new Promise((resolve) => setTimeout(resolve, 100));
    fireEvent.click(trigger);
    expect(await c.findByText(plain.name, {}, { timeout: 3000 })).toBeTruthy();
  });

  it("should report no failure that lands after the picker unmounts", async () => {
    const plain = await createPlain();
    const { unary } = client.transport;
    const send = unary.send.bind(unary);
    const gate = Promise.withResolvers<void>();
    const failed = Promise.withResolvers<void>();
    let lookups = 0;
    const spy = vi.spyOn(unary, "send").mockImplementation(async (...args) => {
      if (args[0] !== LOOKUP) return await send(...args);
      lookups++;
      await gate.promise;
      failed.resolve();
      throw new Error("calculations unavailable");
    });
    onTestFinished(() => {
      gate.resolve();
      spy.mockRestore();
    });
    const reported: { current: Status.NotificationSpec[] } = { current: [] };
    const Reader = (): null => {
      reported.current = Status.useNotifications().statuses;
      return null;
    };
    // Reads the channel on each render, as the list of an open picker does.
    const Probe = ({ ch }: { ch: channel.Channel }): null => {
      Calculation.useNonOutputFilter({ open: true }).filter(ch);
      return null;
    };
    const Host = ({ mounted }: { mounted: boolean }): ReactElement => (
      <>
        <Reader />
        {mounted && <Probe ch={plain} />}
      </>
    );
    const c = render(<Host mounted />, { wrapper });
    await waitFor(() => expect(lookups).toBe(1));
    // The form that holds the open picker closes before the lookup fails.
    c.rerender(<Host mounted={false} />);
    gate.resolve();
    await failed.promise;
    await new Promise((resolve) => setTimeout(resolve, 200));
    c.rerender(<Host mounted={false} />);
    expect(reported.current.map(({ message }) => message)).toEqual([]);
  });
});

describe("useResultStatus breaker", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const setStatus = async (
    calc: calculation.Calculation,
    variant: "error" | "warning",
    message: string,
    description = "",
  ) =>
    await client.statuses.set({
      owner: calculation.resourceID(calc.key),
      name: "Calculation Status",
      variant,
      message,
      description,
      time: TimeStamp.now(),
    });

  const createCalc = async () =>
    await client.calculations.create({ name: id.create(), expression: "return 2" });

  it("should follow a change of the variant alone", async () => {
    const calc = await createCalc();
    await setStatus(calc, "error", "Failed to run calculation", "cannot read a");
    const { result } = renderHook(
      () => Calculation.useResultStatus({ channel: calc.output }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.data?.variant).toEqual("error"));
    await act(
      async () =>
        await setStatus(calc, "warning", "Failed to run calculation", "cannot read a"),
    );
    await waitFor(() => expect(result.current.data?.variant).toEqual("warning"));
  });

  it("should answer with null once the status is deleted", async () => {
    const calc = await createCalc();
    const st = await setStatus(calc, "error", "invalid expression");
    const { result } = renderHook(
      () => Calculation.useResultStatus({ channel: calc.output }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.data?.message).toEqual("invalid expression"));
    await act(async () => await client.statuses.delete(st.key));
    await waitFor(() => {
      expect(result.current.variant).toEqual("success");
      expect(result.current.data).toBeNull();
    });
  });

  it("should answer with null once the calculation is deleted", async () => {
    const calc = await createCalc();
    await setStatus(calc, "error", "invalid expression");
    const { result } = renderHook(
      () => Calculation.useResultStatus({ channel: calc.output }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.data?.message).toEqual("invalid expression"));
    await act(async () => await client.calculations.delete(calc.key));
    await waitFor(() => expect(result.current.data ?? null).toBeNull());
  });

  it("should follow a channel that a calculation writes after the answer", async () => {
    const plain = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const { result } = renderHook(
      () => Calculation.useResultStatus({ channel: plain.key }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.variant).toEqual("success"));
    expect(result.current.data).toBeNull();
    const calc = await act(
      async () =>
        await client.calculations.create({ output: plain.key, expression: "return 2" }),
    );
    await act(async () => await setStatus(calc, "error", "late status"));
    await waitFor(() => expect(result.current.data?.message).toEqual("late status"));
  });

  it("should show a status set before its calculation reaches the outputs", async () => {
    const plain = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const viewer = createTestClient();
    const viewerWrapper = await createAsyncSynnaxWrapper({ client: viewer });
    const { result } = renderHook(
      () => Calculation.useResultStatus({ channel: plain.key }),
      { wrapper: viewerWrapper },
    );
    await waitFor(() => expect(result.current.variant).toEqual("success"));
    const calc = await client.calculations.create({
      output: plain.key,
      expression: "return 2",
    });
    await setStatus(calc, "error", "streamed status");
    await waitFor(() => expect(result.current.data?.message).toEqual("streamed status"));
  });
});
