// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, schematic, type Synnax } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { mockGeometry } from "@synnaxlabs/lyra/testutil";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { color, deep, id } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, beforeEach, describe, expect, it, type Mock, vi } from "vitest";
import { type z } from "zod";

import { Enumeration } from "@/enumeration";
import { Form } from "@/schematic/node/common/form";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.StateIndicatorNodeConfig>();

const client = createTestClient();

const findInput = async (label: string): Promise<HTMLInputElement> =>
  await screen.findByLabelText<HTMLInputElement>(label);

const OPTIONS: schematic.StateMapping[] = [
  { key: "off", name: "Off", value: 0 },
  { key: "on", name: "On", value: 1, color: color.construct("#2e7d32") },
];

describe("Form.States", () => {
  // Stands in for the Console's name modal, which creates the enumeration on Create.
  let prompt: Mock<Enumeration.CreatePrompt>;
  beforeEach(() => {
    prompt = vi.fn(
      async ({ entries }) =>
        (await client.enumerations.create({ name: `Created ${id.create()}`, entries }))
          .key,
    );
  });
  const createWrapper = async (as: Synnax = client): Promise<FC<PropsWithChildren>> => {
    const Base = await createAsyncSynnaxWrapper({ client: as });
    return ({ children }) => (
      <Base>
        <Triggers.Provider>
          <Enumeration.CreatePromptProvider value={(params) => prompt(params)}>
            {children}
          </Enumeration.CreatePromptProvider>
        </Triggers.Provider>
      </Base>
    );
  };
  // Without a box, the dropdown's virtual list shows no items.
  beforeAll(() => mockGeometry(100, 100));
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createWrapper();
  });

  const renderStates = (
    config: Partial<z.input<typeof schematic.stateIndicatorNodeConfigZ>> = {},
    colorHidden = false,
    as: FC<PropsWithChildren> = wrapper,
  ) =>
    renderForm(
      <Form.States
        enumeration={path.enumeration}
        options={path.options}
        label={path.label.label}
        symbolName="State indicator"
        colorHidden={colorHidden}
      />,
      schematic.stateIndicatorNodeConfigZ.parse({
        variant: "state_indicator",
        options: OPTIONS,
        ...config,
      }),
      schematic.stateIndicatorNodeConfigZ,
      as,
    );

  /** Waits for the node to name an enumeration other than the one it named before. */
  const created = async (
    value: ReturnType<typeof renderStates>["value"],
    before?: enumeration.Key,
  ): Promise<enumeration.Enumeration> => {
    let key: enumeration.Key | undefined;
    await waitFor(() => {
      key = value(path.enumeration) as enumeration.Key | undefined;
      if (key == null || key === before) throw new Error("expected a new enumeration");
    });
    return await client.enumerations.retrieve(key as enumeration.Key);
  };

  describe("options", () => {
    it("should edit the node's own options when it names no enumeration", async () => {
      const { value } = renderStates();
      const name = await findInput("Name 1");
      fireEvent.focus(name);
      fireEvent.change(name, { target: { value: "Closed" } });
      fireEvent.blur(name);
      await waitFor(() =>
        expect(
          (value(path.options) as schematic.StateMapping[]).map((o) => o.name),
        ).toEqual(["Closed", "On"]),
      );
    });

    it("should keep every row of a paste into the options", async () => {
      const { value } = renderStates();
      const name = await findInput("Name 1");
      fireEvent.focus(name);
      fireEvent.paste(name, {
        clipboardData: { getData: () => "Closed\t5\nOpen\t6" },
      });
      await waitFor(() =>
        expect(
          (value(path.options) as schematic.StateMapping[]).map((o) => [
            o.name,
            o.value,
          ]),
        ).toEqual([
          ["Closed", 5],
          ["Open", 6],
        ]),
      );
    });

    it("should remove one of the node's own options", async () => {
      const { value } = renderStates();
      fireEvent.click(await screen.findByRole("button", { name: "Remove entry 1" }));
      await waitFor(() =>
        expect(
          (value(path.options) as schematic.StateMapping[]).map((o) => o.key),
        ).toEqual(["on"]),
      );
    });

    it("should move one of the node's own options", async () => {
      const { value } = renderStates();
      fireEvent.keyDown(await screen.findByRole("button", { name: "Move entry 2" }), {
        key: "ArrowUp",
      });
      await waitFor(() =>
        expect(
          (value(path.options) as schematic.StateMapping[]).map((o) => o.key),
        ).toEqual(["on", "off"]),
      );
    });

    it("should keep a named color picked for an option as its color", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const named = await client.colors.create({
        name: `color-${id.create()}`,
        light: "#00FF00",
        group: palette.key,
      });
      const { value } = renderStates();
      fireEvent.click(await screen.findByRole("button", { name: "Pick color 1" }));
      fireEvent.click(await screen.findByRole("button", { name: named.name }));
      await waitFor(() =>
        expect((value(path.options) as schematic.StateMapping[])[0].color).toEqual(
          color.construct("#00FF00"),
        ),
      );
    });

    it("should leave out the Abnormal column of the options", async () => {
      renderStates();
      await findInput("Color 2");
      expect(screen.queryByLabelText("Abnormal 2")).toBeNull();
    });

    it("should leave out the Color column for a node that draws no color", async () => {
      renderStates({}, true);
      await findInput("Name 2");
      expect(screen.queryByLabelText("Color 2")).toBeNull();
    });
  });

  describe("create", () => {
    const openPicker = async (
      c: ReturnType<typeof renderStates>,
      term = "",
    ): Promise<void> => {
      fireEvent.click(await c.findByRole("button", { name: /^Enumeration/ }));
      if (term !== "")
        fireEvent.change(await c.findByPlaceholderText("Search enumerations..."), {
          target: { value: term },
        });
    };

    const createName = (term: string): string =>
      term === "" ? "New enumeration" : `Create enumeration "${term}"`;

    const openCreate = async (
      c: ReturnType<typeof renderStates>,
      term = "",
    ): Promise<void> => {
      await openPicker(c, term);
      fireEvent.click(await c.findByRole("option", { name: createName(term) }));
    };

    it("should lock the options while it creates an enumeration", async () => {
      const c = renderStates();
      await openCreate(c, `Valve ${id.create()}`);
      expect(c.queryByRole("button", { name: "Add entry" })).toBeNull();
      await created(c.value);
    });

    it("should not offer Create enumeration to a user who cannot create", async () => {
      const viewer = await new RoleClients(client).get("Viewer");
      const viewerWrapper = await createWrapper(viewer);
      // A render as a user who is not an admin suspends at first.
      let c: ReturnType<typeof renderStates> | undefined;
      await act(async () => {
        c = renderStates({}, false, viewerWrapper);
      });
      if (c == null) throw new Error("expected a render");
      fireEvent.click(c.getByRole("button", { name: "Enumeration" }));
      expect(await c.findByText("No enumeration")).toBeTruthy();
      await act(async () => {});
      expect(c.queryByRole("option", { name: createName("") })).toBeNull();
    });

    it("should name a new enumeration after the search text", async () => {
      const term = `Valve ${id.create()}`;
      const c = renderStates({ label: { label: "Pump" } });
      await openCreate(c, term);
      const e = await created(c.value);
      expect(e.name).toBe(term);
    });

    it("should move the node's options into a new enumeration", async () => {
      const c = renderStates();
      await openCreate(c);
      const e = await created(c.value);
      expect(
        e.entries.map(({ name, value, color: ref }) => ({ name, value, ref })),
      ).toEqual([
        { name: "Off", value: 0, ref: undefined },
        { name: "On", value: 1, ref: { type: "fixed", value: OPTIONS[1].color } },
      ]);
    });

    it("should start the name of an unlinked node's enumeration with its label", async () => {
      const c = renderStates({ label: { label: "Pump" } });
      await openCreate(c);
      await created(c.value);
      expect(prompt).toHaveBeenCalledWith(expect.objectContaining({ name: "Pump" }));
    });

    it("should start the name with the symbol for a node with no label", async () => {
      const c = renderStates();
      await openCreate(c);
      await created(c.value);
      expect(prompt).toHaveBeenCalledWith(
        expect.objectContaining({ name: "State indicator" }),
      );
    });

    it("should keep the node's options when the user cancels the name", async () => {
      prompt.mockResolvedValueOnce(null);
      const c = renderStates();
      await openCreate(c);
      await act(async () => {});
      expect(prompt).toHaveBeenCalledOnce();
      expect(c.value(path.enumeration)).toBeUndefined();
      expect((await findInput("Name 1")).value).toBe("Off");
    });

    it("should list Create enumeration above No enumeration", async () => {
      const c = renderStates();
      await openPicker(c);
      const create = await c.findByRole("option", { name: createName("") });
      const none = c.getByRole("option", { name: /No enumeration/ });
      expect(
        create.compareDocumentPosition(none) & Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
    });

    it("should not offer to create a name an enumeration has in any case", async () => {
      const e = await client.enumerations.create({
        name: `Valve ${id.create()}`,
        entries: [],
      });
      const c = renderStates();
      const draft = `${e.name} draft`;
      await openPicker(c, draft);
      expect(await c.findByRole("option", { name: createName(draft) })).toBeTruthy();
      fireEvent.change(c.getByPlaceholderText("Search enumerations..."), {
        target: { value: e.name.toUpperCase() },
      });
      await waitFor(() =>
        expect(c.queryByRole("option", { name: /^Create enumeration/ })).toBeNull(),
      );
    });

    it("should create an empty enumeration for a node that names one", async () => {
      const linked = await client.enumerations.create({
        name: id.create(),
        entries: [{ key: id.create(), name: "Primed", value: 0, abnormal: false }],
      });
      const term = `Pump ${id.create()}`;
      const c = renderStates({ enumeration: linked.key });
      await openCreate(c, term);
      const e = await created(c.value, linked.key);
      expect(e.name).toBe(term);
      expect(e.entries).toEqual([]);
      await waitFor(() =>
        expect(document.activeElement).toBe(
          c.getByRole("button", { name: "Add entry" }),
        ),
      );
    });

    it("should ask for the name of a linked node's new enumeration", async () => {
      const linked = await client.enumerations.create({
        name: id.create(),
        entries: [],
      });
      const c = renderStates({ enumeration: linked.key, label: { label: "Pump" } });
      await openCreate(c);
      const e = await created(c.value, linked.key);
      expect(prompt).toHaveBeenCalledWith({ name: "", entries: [] });
      expect(e.entries).toEqual([]);
      await waitFor(() =>
        expect(document.activeElement).toBe(
          c.getByRole("button", { name: "Add entry" }),
        ),
      );
    });

    it("should not focus a created enumeration's entries on a later pick", async () => {
      const linked = await client.enumerations.create({
        name: `Valve ${id.create()}`,
        entries: [],
      });
      const c = renderStates({ enumeration: linked.key });
      await openCreate(c);
      const e = await created(c.value, linked.key);
      const add = (): HTMLElement => c.getByRole("button", { name: "Add entry" });
      await waitFor(() => expect(document.activeElement).toBe(add()));
      await openPicker(c, linked.name);
      fireEvent.click(await c.findByRole("option", { name: linked.name }));
      await waitFor(() => expect(c.value(path.enumeration)).toBe(linked.key));
      await openPicker(c, e.name);
      fireEvent.click(await c.findByRole("option", { name: e.name }));
      await waitFor(() => expect(c.value(path.enumeration)).toBe(e.key));
      await act(async () => {});
      expect(document.activeElement).not.toBe(add());
    });

    it("should make no copy on a second click of a typed create", async () => {
      const linked = await client.enumerations.create({
        name: id.create(),
        entries: [],
      });
      const term = `Pump ${id.create()}`;
      const c = renderStates({ enumeration: linked.key });
      await openCreate(c, term);
      expect(c.queryByRole("option", { name: createName(term) })).toBeNull();
      await created(c.value, linked.key);
      const named = (await client.enumerations.retrieve({ searchTerm: term })).filter(
        (e) => e.name === term,
      );
      expect(named).toHaveLength(1);
    });
  });

  describe("enumeration", () => {
    /** Creates a schematic whose one select draws the enumeration. */
    const createUser = async (e: enumeration.Key): Promise<schematic.Schematic> => {
      const project = await client.projects.create({ name: id.create() });
      return await client.schematics.create({
        project: project.key,
        name: id.create(),
        nodes: [
          {
            key: "select",
            position: { x: 0, y: 0 },
            zIndex: 0,
            config: schematic.nodeConfigZ.parse({ variant: "select", enumeration: e }),
          },
        ],
      });
    };

    it("should count a schematic that links the enumeration while it shows", async () => {
      const e = await client.enumerations.create({ name: id.create(), entries: [] });
      await createUser(e.key);
      const c = renderStates({ enumeration: e.key });
      expect(await c.findByText("1 schematic")).toBeTruthy();
      await createUser(e.key);
      expect(await c.findByText("2 schematics")).toBeTruthy();
    });

    it("should edit the entries of the enumeration that the node names", async () => {
      const e = await client.enumerations.create({
        name: id.create(),
        entries: [{ key: id.create(), name: "Primed", value: 0, abnormal: false }],
      });
      renderStates({ enumeration: e.key });
      expect((await findInput("Name 1")).value).toBe("Primed");
    });

    it("should go back to the node's options on No enumeration", async () => {
      const e = await client.enumerations.create({ name: id.create(), entries: [] });
      const c = renderStates({ enumeration: e.key });
      fireEvent.click(await c.findByRole("button", { name: /^Enumeration/ }));
      fireEvent.click(await c.findByText("No enumeration"));
      await waitFor(() => expect(c.value(path.enumeration)).toBeUndefined());
      expect((await findInput("Name 1")).value).toBe("Off");
    });
  });
});
