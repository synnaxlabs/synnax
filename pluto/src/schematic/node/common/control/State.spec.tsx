// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { control } from "@/telem/control/aether";
import { render } from "@/testutil";

const renderState = (channel?: channel.Key) =>
  render(
    <ReactFlowProvider>
      <Haul.Provider>
        <div data-id="node">
          <Grid.Grid nodeKey="node" editable={false}>
            <Control.State config={Control.stateConfigZ.parse({})} channel={channel} />
          </Grid.Grid>
        </div>
      </Haul.Provider>
    </ReactFlowProvider>,
    { registry: control.REGISTRY },
  );

describe("Control.State", () => {
  it("should show nothing without a channel", () => {
    const { queryAllByRole } = renderState(undefined);
    expect(queryAllByRole("button")).toHaveLength(0);
  });

  it("should show nothing for the zero channel", () => {
    const { queryAllByRole } = renderState(0);
    expect(queryAllByRole("button")).toHaveLength(0);
  });
});
