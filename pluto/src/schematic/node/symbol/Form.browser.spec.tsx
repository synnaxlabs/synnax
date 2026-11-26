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
import { color, deep, id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { Enumeration } from "@/enumeration";
import { Node } from "@/schematic/node";
import { Form } from "@/schematic/node/symbol/Form";
import { createAsyncSynnaxWrapper, pickOption, renderFormSuspended } from "@/testutil";

const path = deep.path<schematic.SymbolNodeConfig>();

const VALVE: schematic.symbol.Ref = { type: "builtin", name: "valve" };

describe("Symbol form", () => {
  const client = createTestClient();
  let prompt: Mock<Enumeration.CreatePrompt>;
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    prompt = vi.fn();
    const Base = await createAsyncSynnaxWrapper({ client });
    wrapper = ({ children }) => (
      <Base>
        <Enumeration.CreatePromptProvider value={(params) => prompt(params)}>
          {children}
        </Enumeration.CreatePromptProvider>
      </Base>
    );
  });
  beforeEach(() => {
    // The user cancels the prompt unless a test says otherwise.
    prompt.mockReset().mockResolvedValue(null);
  });

  const render = async (tab: "telemetry" | "style" = "telemetry") =>
    await renderFormSuspended(
      <Form tab={tab} onTabChange={vi.fn()} />,
      schematic.symbolNodeConfigZ.parse({ variant: "symbol", spec: VALVE }),
      schematic.symbolNodeConfigZ,
      wrapper,
    );

  it("should store a custom color as the node color", async () => {
    const { container, value } = await render("style");
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

  it("should name an enumeration for an enumeration state", async () => {
    const created = await client.enumerations.create({
      name: `Valve ${id.create()}`,
      entries: [{ key: "shut", value: 0, name: "Shut" }],
    });
    const { findByRole, value } = await render();
    fireEvent.click(await findByRole("button", { name: /^Enumeration/ }));
    await pickOption(created.name, "Search enumerations...");
    await waitFor(() =>
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", enumeration: created.key }),
      ]),
    );
  });

  it("should pick the default from the entries of the enumeration", async () => {
    const { findByText, value } = await render();
    fireEvent.click(await findByText("Closed"));
    fireEvent.click(await screen.findByRole("option", { name: "Open" }));
    await waitFor(() =>
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({ key: "active", default: 1 }),
      ]),
    );
  });

  it("should keep the edits made while a new enumeration is created", async () => {
    let link: (key: enumeration.Key) => void = () => {};
    prompt.mockImplementationOnce(
      async () => await new Promise((resolve) => (link = resolve)),
    );
    const { findByRole, findByText, value } = await render();
    fireEvent.click(await findByRole("button", { name: /^Enumeration/ }));
    fireEvent.click(await screen.findByRole("option", { name: "New enumeration" }));
    fireEvent.click(await findByText("Local"));
    const created = await client.enumerations.create({ name: `Valve ${id.create()}` });
    link(created.key);
    await waitFor(() =>
      expect(value(path.states)).toEqual([
        schematic.stateBindingZ.parse({
          key: "active",
          source: { type: "local", value: 0 },
          enumeration: created.key,
        }),
      ]),
    );
  });

  it("should start a new enumeration with the entries that the state draws", async () => {
    const { findByRole, findByText } = await render();
    // The default menu loads the entries of the symbol's enumeration.
    await findByText("Closed");
    fireEvent.click(await findByRole("button", { name: /^Enumeration/ }));
    fireEvent.click(await screen.findByRole("option", { name: "New enumeration" }));
    const [state] = Node.BUILT_INS.get("valve")?.states ?? [];
    if (state?.type !== "enumeration" || state.enumeration == null)
      throw new Error("expected the valve to draw an enumeration");
    const drawn = await client.enumerations.retrieve({ key: state.enumeration });
    await waitFor(() =>
      expect(prompt).toHaveBeenCalledWith({ name: "State", entries: drawn.entries }),
    );
  });
});
