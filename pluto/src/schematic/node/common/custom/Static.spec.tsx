// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { act, render, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Custom } from "@/schematic/node/common/custom";
import { createAsyncSynnaxWrapper } from "@/testutil";

const VALID_SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" /></svg>';

const createData = (svg: string): schematic.symbol.Spec => ({
  drawing: { type: "svg", markup: svg },
  parts: [],
  handles: [],
  states: [],
  commands: [],
  looks: [],
  interactions: [],
  variant: "static",
  scale: 1,
  strokeScaled: false,
  previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
});

describe("Custom.Static", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createSymbol = async (svg: string): Promise<schematic.symbol.Symbol> =>
    await client.schematics.symbols.create({ name: "sym", data: createData(svg) });

  it("should warn when the symbol fails to load", async () => {
    const symbol = await createSymbol(VALID_SVG);
    // A user with no policies cannot read the symbol, so its load fails.
    const denied = await createTestClientWithPolicy(client, []);
    const { findByText } = render(<Custom.Static specKey={symbol.key} />, {
      wrapper: await createAsyncSynnaxWrapper({ client: denied }),
    });
    await findByText("Failed to load symbol");
  });

  it("should warn when the symbol markup does not parse", async () => {
    const symbol = await createSymbol("<svg><rect></svg>");
    const { findByText, container } = render(<Custom.Static specKey={symbol.key} />, {
      wrapper,
    });
    await findByText("Invalid symbol");
    expect(container.querySelector("svg rect")).toBeNull();
  });

  it("should mount the symbol once valid markup replaces it", async () => {
    const symbol = await createSymbol("<svg><rect></svg>");
    const { findByText, queryByText, container } = render(
      <Custom.Static specKey={symbol.key} />,
      { wrapper },
    );
    await findByText("Invalid symbol");
    await act(async () => {
      await client.schematics.symbols.dispatch(
        symbol.key,
        schematic.symbol.set({ path: ["data", "drawing", "markup"], value: VALID_SVG }),
      );
    });
    await waitFor(() => {
      expect(queryByText("Invalid symbol")).toBeNull();
      expect(container.querySelector("svg rect")).not.toBeNull();
    });
  });
});
