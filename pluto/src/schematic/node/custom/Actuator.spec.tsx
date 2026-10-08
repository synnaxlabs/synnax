// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology, type schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, render, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Actuator } from "@/schematic/node/custom/Actuator";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const VALID_SVG = '<svg viewBox="0 0 10 10"><rect width="10" height="10" /></svg>';

const createData = (svg: string): schematic.symbol.Spec => ({
  svg,
  states: [],
  handles: [],
  variant: "static",
  scale: 1,
  strokeScaled: false,
  previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
});

describe("Actuator", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createSymbol = async (
    svg: string,
  ): Promise<{ symbol: schematic.symbol.Symbol; parent: ontology.ID }> => {
    const group_ = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: `custom-Actuator-${Math.random().toString(36).slice(2, 8)}`,
    });
    const parent = group.ontologyID(group_.key);
    const symbol = await client.schematics.symbols.create({
      name: "sym",
      parent,
      data: createData(svg),
    });
    return { symbol, parent };
  };

  it("should warn when the symbol markup does not parse", async () => {
    const { symbol } = await createSymbol("<svg><rect></svg>");
    const { findByText, container } = render(<Actuator specKey={symbol.key} />, {
      wrapper,
    });
    await findByText("Invalid custom symbol");
    expect(container.querySelector("svg rect")).toBeNull();
  });

  it("should mount the symbol once valid markup replaces it", async () => {
    const { symbol, parent } = await createSymbol("<svg><rect></svg>");
    const { findByText, queryByText, container } = render(
      <Actuator specKey={symbol.key} />,
      { wrapper },
    );
    await findByText("Invalid custom symbol");
    await act(async () => {
      await client.schematics.symbols.create({
        key: symbol.key,
        name: symbol.name,
        parent,
        data: createData(VALID_SVG),
      });
    });
    await waitFor(() => {
      expect(queryByText("Invalid custom symbol")).toBeNull();
      expect(container.querySelector("svg rect")).not.toBeNull();
    });
  });
});
