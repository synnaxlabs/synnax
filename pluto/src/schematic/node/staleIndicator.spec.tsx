// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { render, waitFor } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import type { ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Gauge } from "@/schematic/node/general/gauge";
import { Scale } from "@/schematic/node/general/scale";
import { StringDisplay } from "@/schematic/node/general/stringDisplay";
import { Value } from "@/schematic/node/general/value";
import { Tank } from "@/schematic/node/vessels/tank";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { gauge } from "@/vis/gauge/aether";
import { scale } from "@/vis/scale/aether";
import { Staleness } from "@/vis/staleness";
import { stalenessTest } from "@/vis/staleness/aether/testutil";
import { stringValue } from "@/vis/stringValue/aether";
import { value } from "@/vis/value/aether";

const staleColorHidden = vi.fn();
const onUpdate = (state: { staleColorHidden: boolean }): void =>
  staleColorHidden(state.staleColorHidden);

const Wrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: {
    ...stalenessTest.createRegistry(value.Value.TYPE, value.Value.z, onUpdate),
    ...stalenessTest.createRegistry(gauge.Gauge.TYPE, gauge.Gauge.z, onUpdate),
    ...stalenessTest.createRegistry(scale.Scale.TYPE, scale.Scale.z, onUpdate),
    ...stalenessTest.createRegistry(
      stringValue.StringValue.TYPE,
      stringValue.StringValue.z,
      onUpdate,
    ),
  },
  telemFactories: [],
});

const NODE = { nodeKey: "n1", selected: false, onConfigChange: () => {} };

const READOUTS: Array<[string, () => ReactElement]> = [
  [
    "value",
    () => (
      <Value.spec.Node
        {...NODE}
        config={schematic.valueNodeConfigZ.parse({ variant: "value", channel: 1 })}
      />
    ),
  ],
  [
    "gauge",
    () => (
      <Gauge.spec.Node
        {...NODE}
        config={schematic.gaugeNodeConfigZ.parse({ variant: "gauge", channel: 1 })}
      />
    ),
  ],
  [
    "scale",
    () => (
      <Scale.spec.Node
        {...NODE}
        config={schematic.scaleNodeConfigZ.parse({ variant: "scale", channel: 1 })}
      />
    ),
  ],
  [
    "tank",
    () => (
      <Tank.spec.Node
        {...NODE}
        config={schematic.tankNodeConfigZ.parse({ variant: "tank", channel: 1 })}
      />
    ),
  ],
  [
    "string display",
    () => (
      <StringDisplay.spec.Node
        {...NODE}
        config={schematic.stringDisplayNodeConfigZ.parse({
          variant: "string_display",
          channel: 1,
        })}
      />
    ),
  ],
];

describe.each(READOUTS)("Stale.Indicator on a %s", (_, node) => {
  const ui = (badged: boolean): ReactElement => (
    <Wrapper>
      <Staleness.Provider badged={badged}>
        <ReactFlowProvider>{node()}</ReactFlowProvider>
      </Staleness.Provider>
    </Wrapper>
  );

  beforeEach(() => staleColorHidden.mockClear());

  it("should show the indicator and keep the readout's own color", async () => {
    const { findByRole } = render(ui(true));
    const indicator = await findByRole("img", { name: "Stale" });
    expect(indicator.closest(".pluto-grid__item")?.classList).toContain(
      "lyra--location-bottom",
    );
    expect(staleColorHidden).toHaveBeenLastCalledWith(true);
  });

  // Starts badged, so the indicator proves the stale state reached the symbol first.
  it("should paint the stale color with no indicator when not badged", async () => {
    const { findByRole, queryByRole, rerender } = render(ui(true));
    await findByRole("img", { name: "Stale" });
    rerender(ui(false));
    await waitFor(() => expect(staleColorHidden).toHaveBeenLastCalledWith(false));
    expect(queryByRole("img", { name: "Stale" })).toBeNull();
  });
});
