// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { theme } from "@synnaxlabs/lyra/theme";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Custom } from "@/schematic/node/common/custom";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil";

const fixed = (hex: string) => ({
  type: "fixed" as const,
  value: color.construct(hex),
});

const createSpec = (
  spec: Partial<schematic.symbol.New["data"]>,
): schematic.symbol.Spec =>
  schematic.symbol.specZ.parse({
    drawing: { type: "svg", markup: "<svg/>" },
    variant: "static",
    parts: [{ key: "body", name: "Body", selectors: [".body"] }],
    ...spec,
  });

describe("Custom.listValues", () => {
  const entries: enumeration.Entry[] = [
    { key: "closed", value: 0, name: "Closed", abnormal: false },
    { key: "open", value: 1, name: "Open", abnormal: false },
  ];

  it("should list Base and Active for an actuator", () => {
    expect(Custom.listValues(createSpec({ variant: "actuator" }))).toEqual([
      { key: "base", value: 0, name: "Base" },
      { key: "active", value: 1, name: "Active" },
    ]);
  });

  it("should list only Base for a static symbol, whatever its entries", () => {
    expect(Custom.listValues(createSpec({ variant: "static" }), entries)).toEqual([
      { key: "closed", value: 0, name: "Closed" },
    ]);
  });

  it("should name each value after its entry", () => {
    const one = entries.slice(0, 1);
    expect(Custom.listValues(createSpec({ variant: "actuator" }), one)).toEqual([
      { key: "closed", value: 0, name: "Closed" },
      { key: "active", value: 1, name: "Active" },
    ]);
  });
});

describe("Custom.usePaints state values", () => {
  const RED = color.construct("#ff0000");
  const BLUE = color.construct("#0000ff");

  const createValueSpec = () =>
    createSpec({
      states: [
        { type: "enumeration", key: "a", name: "A", default: 1 },
        { type: "enumeration", key: "b", name: "B" },
        {
          type: "number",
          key: "level",
          name: "Level",
          bounds: { lower: 5, upper: 10 },
        },
      ],
      looks: [
        {
          key: "a",
          part: "body",
          when: { type: "equals", state: "a", values: [1] },
          fill: { type: "color", value: fixed("#ff0000") },
        },
        {
          key: "b",
          part: "body",
          when: { type: "equals", state: "b", values: [1] },
          fill: { type: "color", value: fixed("#0000ff") },
        },
        {
          key: "level",
          part: "body",
          when: { type: "equals", state: "level", values: [5] },
          hidden: true,
        },
      ],
    });

  const renderValues = async (
    value: Custom.StateValues,
    spec: schematic.symbol.Spec = createValueSpec(),
  ): Promise<Custom.Paint[]> => {
    const { result } = renderHook(() => Custom.usePaints(spec, value), {
      wrapper: createSynnaxWrapper({ client: null }),
    });
    await waitFor(() => expect(result.current).not.toHaveLength(0));
    return result.current;
  };

  it("should read each state by its key", async () => {
    const values = new Map([
      ["a", 0],
      ["b", 1],
      ["level", 7],
    ]);
    expect(await renderValues(values)).toEqual([{ selectors: [".body"], fill: BLUE }]);
  });

  it("should read a state with no value at its default", async () => {
    expect(await renderValues(new Map([["level", 7]]))).toEqual([
      { selectors: [".body"], fill: RED },
    ]);
  });

  it("should read a NaN value as no value", async () => {
    const values = new Map([
      ["a", NaN],
      ["b", NaN],
      ["level", NaN],
    ]);
    expect(await renderValues(values)).toEqual([
      { selectors: [".body"], fill: RED },
      { selectors: [".body"], hidden: true },
    ]);
  });

  it.each([
    ["no value", new Map<string, number>()],
    ["a NaN value", new Map([["a", NaN]])],
  ])(
    "should read a within state with %s at its default, with no match",
    async (_, values) => {
      const spec = createSpec({
        states: [
          {
            type: "enumeration",
            key: "a",
            name: "A",
            match: { type: "within", bounds: { lower: 50, upper: 100 } },
            default: 1,
          },
        ],
        looks: [
          {
            key: "a",
            part: "body",
            when: { type: "equals", state: "a", values: [1] },
            fill: { type: "color", value: fixed("#ff0000") },
          },
        ],
      });
      expect(await renderValues(values, spec)).toEqual([
        { selectors: [".body"], fill: RED },
      ]);
    },
  );

  it("should read a number state with no value at its lower bound", async () => {
    expect(await renderValues(new Map([["a", 0]]))).toEqual([
      { selectors: [".body"], hidden: true },
    ]);
  });

  it("should read one number as the value of every state", async () => {
    expect(await renderValues(1)).toEqual([
      { selectors: [".body"], fill: RED },
      { selectors: [".body"], fill: BLUE },
    ]);
  });
});
const LIGHT = theme.themeZ.parse(theme.SYNNAX_LIGHT);
const NODE = "#abcdef";
const BANDED_NODE = theme.legible(NODE, LIGHT);

describe("Custom.usePaints", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  const Light: FC<PropsWithChildren> = ({ children }) =>
    createElement(
      wrapper,
      null,
      createElement(Theming.Provider, { theme: theme.SYNNAX_LIGHT }, children),
    );
  let valve: enumeration.Key;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    const created = await client.enumerations.create({
      name: "Paint valve",
      entries: [
        { key: "closed", value: 0, name: "Closed", color: fixed("#ff0000") },
        { key: "open", value: 1, name: "Open", color: fixed("#00ff00") },
        { key: "fault", value: 2, name: "Fault", color: fixed("#000080") },
      ],
    });
    valve = created.key;
  });

  const renderPaints = async (
    spec: schematic.symbol.Spec,
    value: Custom.StateValues,
    options?: Custom.UsePaintsParams,
  ): Promise<Custom.Paint[]> => {
    const { result } = renderHook(() => Custom.usePaints(spec, value, options), {
      wrapper: Light,
    });
    await waitFor(() => expect(result.current).not.toHaveLength(0));
    return result.current;
  };

  describe("useRegions", () => {
    it("should give each part the colors of its looks at the value", async () => {
      const spec = createSpec({
        states: [
          { type: "enumeration", key: "active", name: "Active", enumeration: valve },
        ],
        looks: [
          { key: "fill", part: "body", fill: { type: "entry", state: "active" } },
        ],
      });
      const { result } = renderHook(() => Custom.useRegions(spec, 1), { wrapper });
      await waitFor(() => expect(result.current).toBeDefined());
      expect(result.current).toEqual([
        {
          key: "body",
          name: "Body",
          selectors: [".body"],
          fillColor: color.construct("#00ff00"),
        },
      ]);
    });
  });

  describe("entry paint", () => {
    const createEntrySpec = (match: schematic.symbol.Matcher = { type: "exact" }) =>
      createSpec({
        states: [
          {
            type: "enumeration",
            key: "active",
            name: "Active",
            enumeration: valve,
            match,
          },
        ],
        looks: [
          { key: "fill", part: "body", fill: { type: "entry", state: "active" } },
        ],
      });

    it("should draw the color of the entry that the value selects", async () => {
      expect(await renderPaints(createEntrySpec(), 1)).toEqual([
        { selectors: [".body"], fill: color.construct("#00ff00") },
      ]);
    });

    it("should draw the default entry for a value that no entry has", async () => {
      expect(await renderPaints(createEntrySpec(), 7)).toEqual([
        { selectors: [".body"], fill: color.construct("#ff0000") },
      ]);
    });

    it("should draw the entry of each state at its own value, or at its default", async () => {
      const spec = createSpec({
        states: [
          { type: "enumeration", key: "a", name: "A", enumeration: valve },
          { type: "enumeration", key: "b", name: "B", enumeration: valve, default: 2 },
        ],
        looks: [
          { key: "a", part: "body", fill: { type: "entry", state: "a" } },
          { key: "b", part: "body", fill: { type: "entry", state: "b" } },
        ],
      });
      expect(await renderPaints(spec, new Map([["a", 1]]))).toEqual([
        { selectors: [".body"], fill: color.construct("#00ff00") },
        { selectors: [".body"], fill: color.construct("#000080") },
      ]);
    });

    it.each([
      [1.05, "#00ff00"],
      [0.5, "#ff0000"],
      [1.1, "#ff0000"],
    ])("should read %d through a within match", async (value, hex) => {
      const spec = createEntrySpec({
        type: "within",
        bounds: { lower: 0.9, upper: 1.1 },
      });
      expect(await renderPaints(spec, value)).toEqual([
        { selectors: [".body"], fill: color.construct(hex) },
      ]);
    });
  });

  describe("contrast paint", () => {
    const createContrastSpec = () =>
      createSpec({
        states: [
          { type: "enumeration", key: "active", name: "Active", enumeration: valve },
        ],
        looks: [
          {
            key: "detail",
            part: "body",
            stroke: { type: "contrast", state: "active" },
          },
        ],
      });

    it.each([
      [1, "#000000e6"],
      [2, "#ffffffe6"],
    ])(
      "should draw on entry %d the black or white that reads on it",
      async (v, hex) => {
        expect(await renderPaints(createContrastSpec(), v)).toEqual([
          { selectors: [".body"], stroke: color.construct(hex) },
        ]);
      },
    );
  });

  describe("looks", () => {
    const STROKE = {
      key: "stroke",
      part: "body",
      stroke: { type: "color" as const, value: fixed("#0000ff") },
    };

    it("should apply a look only while its condition holds", async () => {
      const spec = createSpec({
        states: [{ type: "enumeration", key: "active", name: "Active" }],
        looks: [
          STROKE,
          {
            key: "open",
            part: "body",
            when: { type: "equals", state: "active", values: [1] },
            fill: { type: "color", value: fixed("#00ff00") },
          },
        ],
      });
      const stroke = { selectors: [".body"], stroke: color.construct("#0000ff") };
      expect(await renderPaints(spec, 0)).toEqual([stroke]);
      expect(await renderPaints(spec, 1)).toEqual([
        stroke,
        { selectors: [".body"], fill: color.construct("#00ff00") },
      ]);
    });

    it("should draw a look with no part on the whole drawing", async () => {
      const spec = createSpec({ looks: [{ ...STROKE, part: undefined }] });
      expect(await renderPaints(spec, 0)).toEqual([
        { selectors: undefined, stroke: color.construct("#0000ff") },
      ]);
    });

    it("should draw node paint in the theme band of the node color", async () => {
      const spec = createSpec({
        looks: [{ key: "stroke", part: "body", stroke: { type: "node" } }],
      });
      expect(await renderPaints(spec, 0, { nodeColor: NODE })).toEqual([
        { selectors: [".body"], stroke: BANDED_NODE },
      ]);
    });

    it("should draw the override in place of the looks", async () => {
      const spec = createSpec({ looks: [STROKE] });
      const regions = [
        {
          key: "body",
          name: "Body",
          selectors: [".body"],
          strokeColor: color.construct("#123456"),
        },
      ];
      const override = { key: "closed", name: "Closed", regions };
      expect(await renderPaints(spec, 0, { override })).toEqual([
        { selectors: [".body"], stroke: color.construct("#123456") },
      ]);
    });
  });

  describe("theme band", () => {
    const WHITE = color.construct("#ffffff");
    const BANDED_WHITE = theme.legible(WHITE, LIGHT);
    const GREEN = color.construct("#00ff00");

    const createEntryFillSpec = () =>
      createSpec({
        states: [
          { type: "enumeration", key: "active", name: "Active", enumeration: valve },
        ],
        looks: [
          {
            key: "body",
            part: "body",
            stroke: { type: "color", value: fixed("#ffffff") },
            fill: { type: "entry", state: "active" },
          },
        ],
      });

    it("should band an entry fill on a built-in drawing", async () => {
      expect(await renderPaints(createEntryFillSpec(), 1, { builtIn: true })).toEqual([
        {
          selectors: [".body"],
          stroke: BANDED_WHITE,
          fill: theme.legible(GREEN, LIGHT),
        },
      ]);
    });

    it("should keep an entry fill as stored on a custom drawing", async () => {
      expect(await renderPaints(createEntryFillSpec(), 1)).toEqual([
        { selectors: [".body"], stroke: BANDED_WHITE, fill: GREEN },
      ]);
    });

    it("should draw an override that copies the looks the same as the looks", async () => {
      const spec = createEntryFillSpec();
      const { result } = renderHook(() => Custom.useRegions(spec, 1), { wrapper });
      await waitFor(() => expect(result.current).toBeDefined());
      const override = { key: "open", name: "Open", regions: result.current ?? [] };
      expect(await renderPaints(spec, 1, { override })).toEqual(
        await renderPaints(spec, 1),
      );
    });

    it("should draw a color stroke in the band and a color fill as stored", async () => {
      const spec = createSpec({
        looks: [
          {
            key: "white",
            part: "body",
            stroke: { type: "color", value: fixed("#ffffff") },
            fill: { type: "color", value: fixed("#ffffff") },
          },
        ],
      });
      expect(await renderPaints(spec, 0)).toEqual([
        { selectors: [".body"], stroke: BANDED_WHITE, fill: WHITE },
      ]);
    });

    it("should draw an override stroke in the band and its fill as stored", async () => {
      const regions = [
        {
          key: "body",
          name: "Body",
          selectors: [".body"],
          strokeColor: WHITE,
          fillColor: WHITE,
        },
      ];
      const override = { key: "base", name: "Base", regions };
      expect(await renderPaints(createSpec({}), 0, { override })).toEqual([
        { selectors: [".body"], stroke: BANDED_WHITE, fill: WHITE },
      ]);
    });

    const createNoColorSpec = () =>
      createSpec({
        states: [{ type: "enumeration", key: "active", name: "Active" }],
        looks: [
          { key: "fill", part: "body", fill: { type: "entry", state: "active" } },
        ],
      });

    it("should draw the node color for an entry with no color on a built-in drawing", async () => {
      const options = { builtIn: true, nodeColor: NODE };
      expect(await renderPaints(createNoColorSpec(), 0, options)).toEqual([
        { selectors: [".body"], fill: BANDED_NODE },
      ]);
    });

    it("should draw the theme gray for an entry with no color on a built-in drawing", async () => {
      expect(await renderPaints(createNoColorSpec(), 0, { builtIn: true })).toEqual([
        { selectors: [".body"], fill: theme.legible(LIGHT.colors.gray.l11, LIGHT) },
      ]);
    });

    it("should keep the stored fill for an entry with no color on a custom drawing", async () => {
      expect(await renderPaints(createNoColorSpec(), 0)).toEqual([
        { selectors: [".body"] },
      ]);
    });

    it.each([
      [true, BANDED_NODE],
      [false, color.construct(NODE)],
    ])(
      "should band a node fill only on a built-in drawing: %s",
      async (builtIn, fill) => {
        const spec = createSpec({
          looks: [{ key: "fill", part: "body", fill: { type: "node" } }],
        });
        expect(await renderPaints(spec, 0, { builtIn, nodeColor: NODE })).toEqual([
          { selectors: [".body"], fill },
        ]);
      },
    );

    it("should draw contrast on the entry color as a built-in drawing shows it", async () => {
      const spec = createSpec({
        states: [{ type: "enumeration", key: "active", name: "Active" }],
        looks: [
          {
            key: "detail",
            part: "body",
            stroke: { type: "contrast", state: "active" },
          },
        ],
      });
      const options = { builtIn: true, nodeColor: "#ffffff" };
      expect(await renderPaints(spec, 0, options)).toEqual([
        { selectors: [".body"], stroke: color.construct("#ffffffe6") },
      ]);
    });
  });

  describe("useUnmatched", () => {
    // The entry fill shows once the enumeration loads.
    const createStateSpec = () =>
      createSpec({
        states: [
          { type: "enumeration", key: "active", name: "Active", enumeration: valve },
        ],
        looks: [
          { key: "fill", part: "body", fill: { type: "entry", state: "active" } },
        ],
      });

    it.each([
      ["a value that names no entry", new Map([["active", 5]]), true],
      ["a value that names an entry", new Map([["active", 2]]), false],
      ["no value", new Map<string, number>(), false],
      ["one number for every state", 5, false],
    ] as const)("should read %s", async (_, value, unmatched) => {
      const spec = createStateSpec();
      const { result } = renderHook(
        () => ({
          unmatched: Custom.useUnmatched(spec, value),
          paints: Custom.usePaints(spec, value),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.paints).not.toHaveLength(0));
      expect(result.current.unmatched).toBe(unmatched);
    });
  });
});
