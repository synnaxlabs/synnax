// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { aether } from "@synnaxlabs/pluto/ether";
import { TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Log } from "@/feature/log/Log";
import { renderLog } from "@/feature/log/toolbar/testutil";

const stateZ = z.looseObject({ empty: z.boolean().default(true) });
const methodsZ = {
  seek: z.function({ input: z.tuple([z.string()]), output: z.void() }),
};

const seeks = vi.fn();

// The real worker needs a canvas. The stub holds entries so the controls show.
class WorkerStub
  extends aether.Leaf<typeof stateZ, {}, typeof methodsZ>
  implements aether.HandlersFromSchema<typeof methodsZ>
{
  static readonly TYPE = "log";
  schema = stateZ;
  methods = methodsZ;

  afterUpdate(): void {
    if (this.state.empty) this.setState((s) => ({ ...s, empty: false }));
  }

  seek(time: string): void {
    seeks(time);
  }
}

const REGISTRY = { [WorkerStub.TYPE]: WorkerStub };

const enterTime = async (text: string): Promise<void> => {
  fireEvent.click(await screen.findByLabelText("Go to a time"));
  const input = await screen.findByLabelText("Time");
  fireEvent.change(input, { target: { value: text } });
  fireEvent.keyDown(input, { key: "Enter" });
  await act(async () => {});
};

describe("log/Controls", () => {
  beforeEach(() => seeks.mockClear());

  it("should seek to a time read in the log's time zone", async () => {
    await renderLog(Log, { additionalRegistry: REGISTRY, log: { timeZone: "UTC" } });
    await enterTime("2026-08-23 14:05:00");
    const time = TimeStamp.parse("2026-08-23 14:05:00", "UTC");
    expect(seeks).toHaveBeenCalledWith(time?.valueOf().toString());
  });

  it("should not seek to text that is not a time", async () => {
    await renderLog(Log, { additionalRegistry: REGISTRY });
    await enterTime("yesterday-ish");
    expect(seeks).not.toHaveBeenCalled();
    expect(screen.getByLabelText("Time")).toBeDefined();
  });
});
