// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { box, id, scale, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import {
  MAX_ANNOTATIONS,
  Provider,
  type ProviderProps,
} from "@/lineplot/range/aether/provider";
import { renderAether } from "@/testutil/renderAether";
import { render } from "@/vis/render";
import { canvasTest } from "@/vis/render/test";

const REGION = box.construct({ x: 0, y: 0 }, { width: 1000, height: 500 });
const PROVIDER_KEY = "annotations";

/** Supplies the render requestor the provider reads, which production gets from the
 * plot's canvas. A requested render has nothing to schedule: the test drives drawing
 * by calling render itself. */
class Requestor extends aether.Composite<typeof Requestor.stateZ> {
  static readonly TYPE = "range-provider-test-requestor";
  static readonly stateZ = z.object({});
  schema = Requestor.stateZ;

  afterUpdate(ctx: aether.Context): void {
    render.control(ctx, () => {});
  }
}

const renderProps = (timeRange: TimeRange): ProviderProps => ({
  dataToDecimalScale: scale.Scale.scale(
    Number(timeRange.start.valueOf()),
    Number(timeRange.end.valueOf()),
  ),
  region: REGION,
  viewport: REGION,
  timeRange,
});

const mount = (): Provider => {
  const h = renderAether(Requestor, {
    state: {},
    synnax: { props: TEST_CLIENT_PARAMS },
    render: canvasTest.record(),
    registry: { [Provider.TYPE]: Provider },
    children: {
      [PROVIDER_KEY]: {
        type: Provider.TYPE,
        state: { cursor: null, hovered: null, count: 0 },
      },
    },
  });
  return h.child<Provider>(PROVIDER_KEY);
};

describe("Provider", () => {
  // A window over a Core an Arc fills with ranges overlaps far more than the strip can
  // draw, and writing the unbounded answer through the cache stalls the worker for
  // seconds, which freezes every plot in the window.
  it("draws no more annotations than the fetch limit", async () => {
    const client = createTestClient();
    const start = TimeStamp.now();
    const timeRange = new TimeRange(start, start.add(TimeSpan.seconds(10)));
    await client.ranges.create(
      Array.from({ length: MAX_ANNOTATIONS + 20 }, () => ({
        name: `annotation-${id.create()}`,
        timeRange,
        color: "#7C3AED",
      })),
    );
    const provider = mount();
    const props = renderProps(timeRange);
    await expect
      .poll(() => {
        provider.render(props);
        return provider.state.count;
      })
      .toBeGreaterThan(0);
    expect(provider.state.count).toBeLessThanOrEqual(MAX_ANNOTATIONS);
  });
});
