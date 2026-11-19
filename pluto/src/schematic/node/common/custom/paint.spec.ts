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
import { color } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Custom } from "@/schematic/node/common/custom";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

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

describe("Custom.usePaints", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  let valve: enumeration.Key;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    const created = await client.enumerations.create({
      name: "Paint valve",
      entries: [
        { key: "closed", value: 0, name: "Closed", color: fixed("#ff0000") },
        { key: "open", value: 1, name: "Open", color: fixed("#00ff00") },
      ],
    });
    valve = created.key;
  });

  const renderPaints = async (
    spec: schematic.symbol.Spec,
    value: number,
    overrides?: schematic.symbol.StateOverride[],
    nodeColor?: color.Crude,
  ): Promise<Custom.Paint[]> => {
    const { result } = renderHook(
      () => Custom.usePaints(spec, value, overrides, nodeColor),
      { wrapper },
    );
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

    it("should draw node paint in the node color", async () => {
      const spec = createSpec({
        looks: [{ key: "stroke", part: "body", stroke: { type: "node" } }],
      });
      expect(await renderPaints(spec, 0, undefined, "#abcdef")).toEqual([
        { selectors: [".body"], stroke: color.construct("#abcdef") },
      ]);
    });

    it("should draw the override at the value in place of the looks", async () => {
      const spec = createSpec({ looks: [STROKE] });
      const regions = [
        {
          key: "body",
          name: "Body",
          selectors: [".body"],
          strokeColor: color.construct("#123456"),
        },
      ];
      const overrides = [
        { key: "closed", name: "Closed", regions },
        { key: "open", name: "Open", regions: [] },
      ];
      expect(await renderPaints(spec, 0, overrides)).toEqual([
        { selectors: [".body"], stroke: color.construct("#123456") },
      ]);
    });
  });
});
