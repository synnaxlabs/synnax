// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { aetherTest } from "@/aether/test";
import { renderAether } from "@/testutil/renderAether";
import { Diagram } from "@/vis/diagram/aether/Diagram";
import { canvasTest } from "@/vis/render/test";

const REGION = box.construct({ x: 0, y: 0 }, { x: 100, y: 100 });

describe("Diagram", () => {
  it("enqueues a render on initial mount", () => {
    const recorder = canvasTest.record();
    renderAether(Diagram, {
      state: { position: xy.ZERO, zoom: 1, region: REGION, visible: true },
      render: recorder,
    });
    expect(recorder.loopCalls.length).toBeGreaterThan(0);
    expect(recorder.loopCalls[0].args[0]).toMatchObject({
      key: expect.stringContaining(Diagram.TYPE),
      priority: "high",
    });
  });

  it("scissors and rescissors via the render closure on a state change", () => {
    const recorder = canvasTest.record();
    const h = renderAether(Diagram, {
      state: { position: xy.ZERO, zoom: 1, region: REGION, visible: true },
      render: recorder,
    });
    recorder.clear();
    h.setState((p) => ({ ...p, zoom: 2 }));
    expect(recorder.loopCalls.length).toBeGreaterThan(0);
  });

  it("skips view-scale recomputation when visible stays false across updates", () => {
    const recorder = canvasTest.record();
    const h = renderAether(Diagram, {
      state: { position: xy.ZERO, zoom: 1, region: REGION, visible: false },
      render: recorder,
    });
    recorder.clear();
    h.setState((p) => ({ ...p, position: { x: 10, y: 10 } }));
    expect(recorder.loopCalls).toHaveLength(0);
  });

  it("clears the auto-render interval on unmount", () => {
    const recorder = canvasTest.record();
    const h = renderAether(Diagram, {
      state: {
        position: xy.ZERO,
        zoom: 1,
        region: REGION,
        visible: true,
        autoRenderInterval: 100,
      },
      render: recorder,
    });
    recorder.clear();
    h.unmount();
    expect(recorder.loopCalls.length).toBeGreaterThan(0);
  });

  it("renders registered children when its render closure is invoked", () => {
    const recorder = canvasTest.record();
    const h = renderAether(Diagram, {
      state: { position: xy.ZERO, zoom: 1, region: REGION, visible: true },
      render: recorder,
      children: {
        e1: { type: aetherTest.TestComposite.TYPE, state: {} },
      },
    });
    expect(h.child<aetherTest.TestComposite>("e1").updateCalls).toHaveLength(1);
  });
});
