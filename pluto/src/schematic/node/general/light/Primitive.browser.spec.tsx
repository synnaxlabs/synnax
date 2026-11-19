// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { computed, renderThemed } from "@synnaxlabs/lyra/styletest";
import { color } from "@synnaxlabs/x";
import { type CSSProperties } from "react";
import { describe, expect, it } from "vitest";

import { Light } from "@/schematic/node/general/light/Primitive";

const STROKE = color.construct("#3e8ef7");
const ON = color.construct("#dc136c");

const lampFill = async (props: { onColor?: color.Color }): Promise<string> => {
  const { container } = await renderThemed(
    <Light enabled strokeColor={STROKE} {...props} />,
  );
  const lamp = container.querySelector(".pluto-light__lamp circle");
  if (lamp == null) throw new Error("expected a lamp");
  return computed(lamp, "fill");
};

// display resolves c through the theme, as a colored shape draws it.
const display = async (c: color.Color): Promise<string> => {
  const style = { "--pluto-symbol-color": color.rgbaString(c) } as CSSProperties;
  const { container } = await renderThemed(
    <svg className="lyra-symbol-colored" style={style} />,
  );
  const probe = container.querySelector("svg");
  if (probe == null) throw new Error("expected a probe");
  return computed(probe, "stroke");
};

describe("light symbol in a browser", () => {
  it("should fill a lit lamp with the on color", async () => {
    expect(await display(ON)).not.toBe(await display(STROKE));
    expect(await lampFill({ onColor: ON })).toBe(await display(ON));
  });

  it("should fill a lit lamp with the stroke color when there is no on color", async () => {
    expect(await lampFill({})).toBe(await display(STROKE));
  });
});
