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
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Open } from "@/feature/channel/open";
import { createTestSink, wrapWithSink } from "@/platform/analytics/testutil";
import { createResource } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();
const other = createTestClient();

const renderOpen = async () => {
  const proj = await client.projects.create({ name: uniqueName("proj") });
  const { wrapper, store } = await createConsoleWrapper({ client });
  store.dispatch(Session.Project.select(proj.key));
  const analytics = createTestSink();
  const { result } = renderHook(Open.use, { wrapper: wrapWithSink(wrapper, analytics) });
  const open = (ch: channel.Channel) =>
    act(() => result.current(createResource(channel.resourceID(ch.key), ch.name)));
  return { open, analytics };
};

describe("Open.use", () => {
  it("should report the line plot it created for a channel", async () => {
    const ch = await client.channels.create({
      name: uniqueName("ch"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const { open, analytics } = await renderOpen();
    open(ch);
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("plot_created", {}),
    );
  });

  it("should open a virtual channel that no calculation writes in a log", async () => {
    const ch = await client.channels.create({
      name: uniqueName("plain"),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const logs = vi.spyOn(client.logs, "create");
    const { open, analytics } = await renderOpen();
    open(ch);
    await waitFor(() => expect(logs).toHaveBeenCalled());
    expect(analytics.capture).not.toHaveBeenCalledWith("plot_created", {});
  });

  it("should plot a channel that a calculation writes after the cache answered", async () => {
    const ch = await client.channels.create({
      name: uniqueName("later"),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    expect(await client.calculations.outputs.retrieve([ch.key])).toEqual([
      { key: ch.key, hidden: false, calculation: null },
    ]);
    const calc = await other.calculations.create({
      output: ch.key,
      expression: "return 2",
    });
    await waitFor(() =>
      expect(client.calculations.outputs.get(ch.key)).toMatchObject({
        calculation: calc.key,
      }),
    );
    const { open, analytics } = await renderOpen();
    open(ch);
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("plot_created", {}),
    );
  });
});
