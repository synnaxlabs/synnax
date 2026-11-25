// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color, deep } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Form } from "@/schematic/node/symbol/Form";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.SymbolNodeConfig>();

describe("Symbol form style", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should store a custom color as the node color", async () => {
    const { container, value } = renderForm(
      <Form tab="style" onTabChange={vi.fn()} />,
      schematic.symbolNodeConfigZ.parse({
        variant: "symbol",
        spec: { type: "builtin", name: "valve" },
      }),
      schematic.symbolNodeConfigZ,
      wrapper,
    );
    const swatch = container.querySelector<HTMLElement>(".lyra-color-swatch");
    if (swatch == null) throw new Error("no swatch rendered");
    fireEvent.click(swatch);
    fireEvent.click(await screen.findByText("Custom"));
    fireEvent.change(screen.getByLabelText("Hex"), { target: { value: "0000ff" } });
    await waitFor(() =>
      expect(value(path.color)).toEqual({
        type: "fixed",
        value: color.construct("#0000ff"),
      }),
    );
  });
});
