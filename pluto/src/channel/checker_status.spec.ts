// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Channel } from "@/channel";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

// The two specs that 8f-2 deletes from queries.spec.ts, unchanged, while the 8f-2 tree
// still reads a calculated channel's status through Channel.useResultStatus.
describe("checker 8f-2: deleted calculated status specs", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should retrieve a calculated channel with its error status", async () => {
    const source = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const calc = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
      expression: `return ${source.name} * 2`,
    });
    await sleep.sleep(TimeSpan.milliseconds(500));
    await client.statuses.set({
      owner: channel.resourceID(calc.key),
      name: calc.name,
      variant: "error",
      message: "invalid expression",
      time: TimeStamp.now(),
      details: { channel: calc.key },
    });
    const { result } = await renderHookSuspended(() => Channel.use({ key: calc.key }), {
      wrapper,
    });
    await waitFor(() => {
      expect(result.current).not.toBeNull();
      expect(result.current?.status?.variant).toEqual("error");
      expect(result.current?.status?.message).toEqual("invalid expression");
    });
  });

  it("should react to status changes on a calculated channel", async () => {
    const source = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const calc = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
      expression: `return ${source.name} * 2`,
    });
    const { result } = await renderHookSuspended(() => Channel.use({ key: calc.key }), {
      wrapper,
    });
    await waitFor(() => expect(result.current).not.toBeNull());
    expect(result.current?.status).toBeUndefined();
    await sleep.sleep(TimeSpan.milliseconds(500));
    await act(async () => {
      await client.statuses.set({
        owner: channel.resourceID(calc.key),
        name: calc.name,
        variant: "error",
        message: "broken expression",
        time: TimeStamp.now(),
        details: { channel: calc.key },
      });
    });
    await waitFor(() => {
      expect(result.current?.status?.variant).toEqual("error");
      expect(result.current?.status?.message).toEqual("broken expression");
    });
  });
});
