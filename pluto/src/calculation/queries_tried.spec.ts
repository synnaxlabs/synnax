// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import { createTestClient, spyOnSend } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/lyra/icon";
import { id, sleep, TimeSpan } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Calculation } from "@/calculation";
import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const lookups = (send: ReturnType<typeof spyOnSend>): number =>
  send.mock.calls.filter(([target]) => target === "/calculation/retrieve").length;

describe("Calculation queries breaker", () => {
  const client = createTestClient();
  const other = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createVirtual = async () =>
    await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });

  it("should look up many channels of one render in one request", async () => {
    const channels = await Promise.all(Array.from({ length: 12 }, createVirtual));
    const send = spyOnSend(client);
    const { result } = renderHook(
      () => channels.map((ch) => Calculation.useIsOutput(ch.key)),
      { wrapper },
    );
    await sleep.sleep(TimeSpan.milliseconds(300));
    expect(result.current).toEqual(channels.map(() => false));
    expect(lookups(send)).toBe(1);
  });

  it("should ask nothing for the key 0", async () => {
    const send = spyOnSend(client);
    const { result } = renderHook(() => Calculation.useIsOutput(0), { wrapper });
    await sleep.sleep(TimeSpan.milliseconds(200));
    expect(result.current).toBe(false);
    expect(lookups(send)).toBe(0);
  });

  it("should answer false for a key that names no channel", async () => {
    const { result } = renderHook(
      () => Calculation.useResultOutput({ channel: 0xfffff }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.variant).toEqual("success"));
    expect(result.current.data?.calculation).toBeNull();
  });

  it("should show the calculation icon once another client makes a channel an output", async () => {
    const source = await createVirtual();
    const ch = await createVirtual();
    const { result } = renderHook(() => Channel.useKind({ key: ch.key }), { wrapper });
    await waitFor(() => expect(result.current?.icon).toBe(Icon.Decimal));
    await other.calculations.create({
      output: ch.key,
      expression: `return ${source.name} * 2`,
    });
    await waitFor(() => expect(result.current?.icon).toBe(Icon.Calculation));
  });

  it("should drop the calculation icon once another client deletes the calculation", async () => {
    const source = await createVirtual();
    const calc = await client.calculations.create({
      name: id.create(),
      expression: `return ${source.name} * 2`,
    });
    const { result } = renderHook(() => Channel.useKind({ key: calc.output }), {
      wrapper,
    });
    await waitFor(() => expect(result.current?.icon).toBe(Icon.Calculation));
    await other.calculations.delete(calc.key);
    await waitFor(() => expect(result.current?.icon).not.toBe(Icon.Calculation));
  });
});
