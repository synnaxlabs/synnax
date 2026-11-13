// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { Text } from "@synnaxlabs/lyra/text";
import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Primitive } from "@/schematic/node/common/primitive";
import { SIZE_LEVELS } from "@/schematic/node/common/size";
import { Button } from "@/schematic/node/general/button";
import { Input } from "@/schematic/node/general/input";
import { Select } from "@/schematic/node/general/select";
import { Setpoint } from "@/schematic/node/general/setpoint";

interface SymbolProps {
  size: Component.Size;
  level?: text.Level;
}

const SYMBOLS: Record<string, (props: SymbolProps) => ReactElement> = {
  button: (props) => (
    <Button.spec.Preview
      {...schematic.buttonNodeConfigZ.parse({
        ...props,
        variant: "button",
        label: { label: "Text" },
      })}
    />
  ),
  input: (props) => (
    <Input.spec.Preview
      {...schematic.inputNodeConfigZ.parse({ ...props, variant: "input" })}
    />
  ),
  select: (props) => (
    <Select.spec.Preview
      {...schematic.selectNodeConfigZ.parse({ ...props, variant: "select" })}
    />
  ),
  setpoint: (props) => (
    <Setpoint.spec.Preview
      {...schematic.setpointNodeConfigZ.parse({ ...props, variant: "setpoint" })}
    />
  ),
};

const FIELD_SELECTORS: Record<string, string> = {
  button: ".lyra-btn",
  input: "input",
  select: ".lyra-btn",
  setpoint: "input",
};

const fontSize = (el: Element): number => parseFloat(getComputedStyle(el).fontSize);

const measure = async (symbol: string, props: SymbolProps): Promise<number[]> => {
  const { container } = await renderThemed(SYMBOLS[symbol](props));
  const field = container.querySelector(FIELD_SELECTORS[symbol]);
  if (field == null) throw new Error(`expected a ${symbol} field`);
  if (symbol === "button") return [fontSize(field)];
  const send = [...container.querySelectorAll("button")].find(
    (b) => b.textContent === "Set",
  );
  if (send == null) throw new Error(`expected a ${symbol} send button`);
  return [fontSize(field), fontSize(send)];
};

const STANDARD_PX: [Component.Size, number][] = [
  ["tiny", 12],
  ["small", 14],
  ["medium", 16],
  ["large", 21],
  ["huge", 27],
];

const SAVED_PX: [Component.Size, number][] = [
  ["tiny", 11],
  ["small", 12],
  ["medium", 13],
  ["large", 14],
  ["huge", 27],
];

describe("symbol text scale", () => {
  describe.each(Object.keys(SYMBOLS))("%s", (symbol) => {
    it.each(STANDARD_PX)(
      "should render %s at %i px on the standard scale",
      async (size, px) => {
        const sizes = await measure(symbol, { size, level: SIZE_LEVELS[size] });
        sizes.forEach((measured) => expect(measured).toBeCloseTo(px, 0));
      },
    );

    it.each(SAVED_PX)(
      "should keep a saved %s at %i px at its migrated level",
      async (size, px) => {
        const sizes = await measure(symbol, {
          size,
          level: Component.SIZE_TEXT_LEVELS[size],
        });
        sizes.forEach((measured) => expect(measured).toBeCloseTo(px, 0));
      },
    );
  });

  it.each(STANDARD_PX)("should render a %s label at %i px", async (size, px) => {
    const { getByText } = await renderThemed(
      <Text.Text level={SIZE_LEVELS[size]}>Label</Text.Text>,
    );
    expect(fontSize(getByText("Label").element())).toBeCloseTo(px, 0);
  });

  describe("readout", () => {
    const SCALES = Component.SIZES.flatMap((size): [Component.Size, text.Level][] => [
      [size, SIZE_LEVELS[size]],
      [size, Component.SIZE_TEXT_LEVELS[size]],
    ]);

    it.each(SCALES)(
      "should fit %s text inside the readout at field level %s",
      async (size, level) => {
        const { container } = await renderThemed(
          <Primitive.Readout size={size} level={level} value="100.0" />,
        );
        const readout = container.querySelector<HTMLElement>(
          ".pluto-symbol-field__readout",
        );
        const value = readout?.querySelector("p, h3, h4, small");
        if (readout == null || value == null) throw new Error("expected a readout");
        const outer = readout.getBoundingClientRect();
        const inner = value.getBoundingClientRect();
        expect(inner.top).toBeGreaterThanOrEqual(outer.top);
        expect(inner.bottom).toBeLessThanOrEqual(outer.bottom);
      },
    );
  });
});
