// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color, id, TimeSpan } from "@synnaxlabs/x";
import { render, screen, waitFor } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { type FC } from "react";
import { describe, expect, it } from "vitest";

import { aether } from "@/aether/aether";
import { Node } from "@/schematic/node";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";
import { Staleness } from "@/vis/staleness";
import { toggle } from "@/vis/toggle/aether";

const SynnaxWrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: toggle.REGISTRY,
  telemFactories: [],
});

describe("createToggle", () => {
  it("should hold a valve for its stored delay in nanoseconds", () => {
    const Valve = Node.resolveSpec("valve").Node;
    const config = Node.createConfig({
      variant: "valve",
      onClickDelay: Number(TimeSpan.milliseconds(500).valueOf()),
    });
    const { container } = render(
      <SynnaxWrapper>
        <ReactFlowProvider>
          <Valve
            nodeKey="v1"
            selected={false}
            config={config}
            onConfigChange={() => {}}
          />
        </ReactFlowProvider>
      </SynnaxWrapper>,
    );
    const button = container.querySelector("button");
    expect(button?.style.getPropertyValue(Primitive.TOGGLE_DELAY)).toBe("0.5s");
  });
});

// Stands in for the toggle worker component and reports its source as stale.
describe("deleted channel", () => {
  const client = createTestClient();
  const createChannel = async (): Promise<channel.Channel> =>
    await client.channels.create({
      name: `toggle_${id.create()}`,
      dataType: DataType.UINT8,
      virtual: true,
    });

  // The control chip is hidden, as it needs a schematic's controller.
  const renderValve = async (
    stateChannel: channel.Key,
    commandChannel: channel.Key,
  ) => {
    const Providers = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: toggle.REGISTRY,
      telemFactories: [],
    });
    const Valve = Node.resolveSpec("valve").Node;
    const config = Node.createConfig({
      variant: "valve",
      stateChannel,
      commandChannel,
    });
    return render(
      <Providers>
        <ReactFlowProvider>
          <Valve
            nodeKey={id.create()}
            selected={false}
            config={{ ...config, control: { ...config.control, hidden: true } }}
            onConfigChange={() => {}}
          />
        </ReactFlowProvider>
      </Providers>,
    );
  };

  it("should disable and badge a valve whose state channel was deleted", async () => {
    const [state, command] = await Promise.all([createChannel(), createChannel()]);
    await client.channels.delete(state.key);
    const { container } = await renderValve(state.key, command.key);
    await waitFor(() => expect(screen.getByText("Deleted channel")).toBeTruthy());
    expect(container.querySelector("button")?.disabled).toBe(true);
  });

  it("should disable and badge a valve whose command channel was deleted", async () => {
    const [state, command] = await Promise.all([createChannel(), createChannel()]);
    await client.channels.delete(command.key);
    const { container } = await renderValve(state.key, command.key);
    await waitFor(() => expect(screen.getByText("Deleted channel")).toBeTruthy());
    expect(container.querySelector("button")?.disabled).toBe(true);
  });

  it("should leave a valve whose channels exist enabled and unbadged", async () => {
    const [state, command] = await Promise.all([createChannel(), createChannel()]);
    const { container } = await renderValve(state.key, command.key);
    await waitFor(() => expect(container.querySelector("button")).not.toBeNull());
    expect(screen.queryByText("Deleted channel")).toBeNull();
    expect(container.querySelector("button")?.disabled).toBe(false);
  });
});

class StaleToggle extends aether.Leaf<typeof toggle.toggleStateZ> {
  static readonly TYPE = toggle.Toggle.TYPE;
  schema = toggle.toggleStateZ;

  afterUpdate(): void {
    if (!this.state.stale) this.setState((s) => ({ ...s, stale: true }));
  }
}

const StaleWrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: { [StaleToggle.TYPE]: StaleToggle },
  telemFactories: [],
});

const SYMBOL_COLOR = "#ff0000";

const Stub: FC<Omit<schematic.ToggleSymbolConfig, "label"> & Toggle.ButtonProps> = ({
  strokeColor,
}) => <span data-testid="symbol" data-stroke={color.hex(strokeColor ?? "#000000")} />;

const StubToggle = Toggle.createToggle<schematic.ToggleSymbolConfig>(Stub);

const renderStale = (badged: boolean) =>
  render(
    <StaleWrapper>
      <Staleness.Provider badged={badged}>
        <ReactFlowProvider>
          <StubToggle
            nodeKey="v1"
            selected={false}
            config={schematic.toggleSymbolConfigZ.parse({
              stateChannel: 1,
              strokeColor: SYMBOL_COLOR,
            })}
            onConfigChange={() => {}}
          />
        </ReactFlowProvider>
      </Staleness.Provider>
    </StaleWrapper>,
  );

describe("createToggle staleness", () => {
  it("should show the stale badge and keep the symbol color when badged", async () => {
    const { findByRole, getByTestId } = renderStale(true);
    await findByRole("img", { name: "Stale" });
    expect(getByTestId("symbol").dataset.stroke).toBe(SYMBOL_COLOR);
  });

  it("should paint the stale color with no badge when not badged", async () => {
    const { getByTestId, queryByRole } = renderStale(false);
    await waitFor(() =>
      expect(getByTestId("symbol").dataset.stroke).not.toBe(SYMBOL_COLOR),
    );
    expect(queryByRole("img", { name: "Stale" })).toBeNull();
  });
});
