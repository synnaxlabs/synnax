// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { Light } from "@/schematic/node/general/light";
import { createSynnaxWrapper } from "@/testutil";
import { light } from "@/vis/light/aether";
import { stalenessTest } from "@/vis/staleness/aether/testutil";

const COLOR_VAR = "--pluto-symbol-color";
const ON_COLOR_VAR = "--pluto-light-on-color";
const STROKE = "0, 255, 0, 1";
const ON = "255, 0, 0, 1";

const Wrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: stalenessTest.createRegistry(light.Light.TYPE, light.stateZ),
  telemFactories: [],
});

const renderStale = () => {
  const result = render(
    <Wrapper>
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
    </Wrapper>,
  );
  const color = (selector: string, variable: string): string =>
    result.container
      .querySelector<HTMLElement | SVGElement>(selector)
      ?.style.getPropertyValue(variable) ?? "";
  return {
    ...result,
    strokeColor: () => color(".pluto-light svg", COLOR_VAR),
    lampColor: () => color(".pluto-light", ON_COLOR_VAR),
  };
};

describe("Light.Symbol staleness", () => {
  it("should show the stale indicator and keep the symbol colors", async () => {
    const { findByRole, strokeColor, lampColor } = renderStale();
    const indicator = await findByRole("img", { name: "Stale" });
    expect(indicator.closest(".pluto-grid__item")?.classList).toContain(
      "lyra--location-bottom",
    );
    expect(strokeColor()).toBe(STROKE);
    expect(lampColor()).toBe(ON);
  });
});
