// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { waitFor } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { aether } from "@/aether/aether";
import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { control } from "@/telem/control/aether";
import { controlTest } from "@/telem/control/aether/testutil";
import { render } from "@/testutil";

interface RenderOptions {
  config?: Control.StateConfig;
  registry?: aether.ComponentRegistry;
  children?: ReactNode;
}

const renderState = (
  channels: channel.Key[],
  {
    config = Control.stateConfigZ.parse({}),
    registry = control.REGISTRY,
    children,
  }: RenderOptions = {},
) =>
  render(
    <ReactFlowProvider>
      <Haul.Provider>
        <div data-id="node">
          <Grid.Grid nodeKey="node" editable={false}>
            <Control.State config={config} channels={channels}>
              {children}
            </Control.State>
          </Grid.Grid>
        </div>
      </Haul.Provider>
    </ReactFlowProvider>,
    { registry },
  );

describe("Control.State", () => {
  it("should show nothing for the zero channel", () => {
    const { queryAllByRole } = renderState([0]);
    expect(queryAllByRole("button")).toHaveLength(0);
  });

  it("should show its children for the zero channel", () => {
    const { getByText, queryAllByRole } = renderState([0], {
      children: <span>child</span>,
    });
    expect(getByText("child")).toBeTruthy();
    expect(queryAllByRole("button")).toHaveLength(0);
  });
});

const chipSinks = vi.fn();
const STUB_REGISTRY = controlTest.createRegistry(chipSinks);

describe("Control.State authority", () => {
  beforeEach(() => {
    chipSinks.mockClear();
  });

  it("should acquire control with absolute authority by default", async () => {
    renderState([7], { registry: STUB_REGISTRY });
    await waitFor(() =>
      expect(chipSinks).toHaveBeenCalledWith(
        control.acquireChannelControl({ channels: [7], authority: 255 }),
      ),
    );
  });

  it("should acquire control of every channel but the zero channel", async () => {
    renderState([7, 0, 8], { registry: STUB_REGISTRY });
    await waitFor(() =>
      expect(chipSinks).toHaveBeenCalledWith(
        control.acquireChannelControl({ channels: [7, 8], authority: 255 }),
      ),
    );
  });

  it("should acquire control with the configured authority", async () => {
    renderState([7], {
      config: Control.stateConfigZ.parse({ authority: 100 }),
      registry: STUB_REGISTRY,
    });
    await waitFor(() =>
      expect(chipSinks).toHaveBeenCalledWith(
        control.acquireChannelControl({ channels: [7], authority: 100 }),
      ),
    );
  });
});
