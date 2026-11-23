// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SyntheticCore } from "@synnaxlabs/client/testutil";
import { Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import {
  COLOR,
  CommandFactory,
  createControlStubs,
  createWrapper,
  FILLED,
  TestNode,
  WITHIN,
} from "@/schematic/node/symbol/testutil";

const VALVE = { type: "builtin" as const, name: "valve" };
const BUTTON = "button.pluto-symbol";

describe("Symbol node command menu", () => {
  it("should open a menu of the commands when the value equals neither", async () => {
    const onConfigChange = vi.fn();
    const Wrapper = createWrapper();
    const { container, findByRole } = render(
      <Wrapper>
        <TestNode
          config={{
            spec: VALVE,
            states: [
              {
                key: "active",
                source: { type: "local", value: 2 },
                match: { type: "exact" },
              },
            ],
          }}
          onConfigChange={onConfigChange}
        />
      </Wrapper>,
    );
    fireEvent.click(container.querySelector(BUTTON)!);
    fireEvent.click(await findByRole("menuitem", { name: "Open" }));
    expect(onConfigChange).toHaveBeenCalledWith({
      states: [expect.objectContaining({ source: { type: "local", value: 1 } })],
    });
  });

  it("should open the menu while the command channel differs from the state", async () => {
    const now = TimeStamp.now();
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now, channels: 2 });
    // data_0 reads near 0.91 and data_1 near 0.14, so the valve match reads the state
    // as open and its command as closed.
    const [state, command] = core.data;
    const commands = new CommandFactory();
    const { registry } = createControlStubs();
    const Wrapper = createWrapper({ core, commands, registry });
    const { container, findByRole } = render(
      <Wrapper>
        <TestNode
          config={{
            spec: VALVE,
            color: COLOR,
            states: [
              {
                key: "active",
                source: { type: "channel", key: state.key, command: command.key },
                match: WITHIN,
              },
            ],
          }}
        />
      </Wrapper>,
    );
    await act(
      async () => await vi.waitFor(() => expect(core.streams(command.key)).toBe(true)),
    );
    await act(async () => core.advance(TimeSpan.seconds(1)));
    await vi.waitFor(() => expect(container.querySelector(FILLED)).not.toBeNull());
    fireEvent.click(container.querySelector(BUTTON)!);
    expect(await findByRole("menuitem", { name: "Close" })).toBeTruthy();
    expect(commands.sinks.get(command.key)?.values).toEqual([]);
  });
});
