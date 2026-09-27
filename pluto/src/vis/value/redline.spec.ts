// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { telem } from "@/telem/aether";
import { backgroundTelem, type Band } from "@/vis/value/redline";

const SOURCE = telem.fixedString("10");
const RED = color.construct("#ff0000");

const band = (flashing: boolean): Band => ({
  key: "a",
  threshold: 0,
  color: RED,
  flashing,
});

describe("backgroundTelem", () => {
  it("should paint nothing with no bands and no background", () => {
    expect(backgroundTelem(SOURCE, { bands: [], smooth: false })).toBeUndefined();
  });

  it("should paint the background alone when there are no bands", () => {
    expect(backgroundTelem(SOURCE, { bands: [], smooth: false }, RED)).toEqual(
      telem.fixedColor(RED),
    );
  });

  it("should fall back to the background below every band", () => {
    const spec = backgroundTelem(SOURCE, { bands: [band(false)], smooth: false }, RED);
    expect(spec?.props.segments.band.props).toMatchObject({ background: RED });
  });

  it("should wire a clock into the bands when one flashes", () => {
    const spec = backgroundTelem(SOURCE, { bands: [band(true)], smooth: false });
    expect(spec?.props.segments).toHaveProperty("phase");
    expect(spec?.props.connections).toContainEqual({ from: "phase", to: "band" });
  });

  it("should leave the clock out when no band flashes", () => {
    const spec = backgroundTelem(SOURCE, { bands: [band(false)], smooth: false });
    expect(spec?.props.segments).not.toHaveProperty("phase");
  });
});
