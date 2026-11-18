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
import { describe, expect, it } from "vitest";

import { aether } from "@/aether/aether";
import { Light } from "@/schematic/node/general/light";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { light } from "@/vis/light/aether";
import { Staleness } from "@/vis/staleness";

const COLOR_VAR = "--pluto-symbol-color";
const STROKE = "0, 255, 0, 1";
const ON = "255, 0, 0, 1";

// Stands in for the light worker component and reports its source as stale.
class StaleLight extends aether.Leaf<typeof light.stateZ> {
  static readonly TYPE = light.Light.TYPE;
  schema = light.stateZ;

  afterUpdate(): void {
    if (!this.state.stale) this.setState((s) => ({ ...s, stale: true }));
  }
}

const Wrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: { [StaleLight.TYPE]: StaleLight },
  telemFactories: [],
});

const renderStale = (badged: boolean) => {
  const result = render(
    <Wrapper>
      <Staleness.Provider badged={badged}>
        <ReactFlowProvider>
          <Light.spec.Node
            nodeKey="l1"
            selected={false}
            onConfigChange={() => {}}
            config={schematic.lightNodeConfigZ.parse({
              variant: "light",
              channel: 1,
              strokeColor: "#00ff00",
              onColor: "#ff0000",
            })}
          />
        </ReactFlowProvider>
      </Staleness.Provider>
    </Wrapper>,
  );
  const color = (selector: string): string =>
    result.container
      .querySelector<HTMLElement | SVGElement>(selector)
      ?.style.getPropertyValue(COLOR_VAR) ?? "";
  return {
    ...result,
    strokeColor: () => color(".pluto-light svg"),
    lampColor: () => color(".pluto-light__lamp"),
  };
};

describe("Light.Symbol staleness", () => {
  it("should show the stale indicator in place of the stale color", async () => {
    const { findByRole, strokeColor, lampColor } = renderStale(true);
    const indicator = await findByRole("img", { name: "Stale" });
    expect(indicator.closest(".pluto-grid__item")?.classList).toContain(
      "lyra--location-bottom",
    );
    expect(strokeColor()).toBe(STROKE);
    expect(lampColor()).toBe(ON);
  });

  it("should paint the stale color with no indicator when not badged", async () => {
    const { queryByRole, strokeColor, lampColor } = renderStale(false);
    await waitFor(() => expect(strokeColor()).not.toBe(STROKE));
    expect(lampColor()).toBe("");
    expect(queryByRole("img", { name: "Stale" })).toBeNull();
  });
});
