// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, schematic } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { color, deep, id } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";
import { type z } from "zod";

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
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    const Base = await createAsyncSynnaxWrapper({ client });
    wrapper = ({ children }) => (
      <Base>
        <Triggers.Provider>{children}</Triggers.Provider>
      </Base>
    );
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

    /** Presses Create in the modal that asks for the name. */
    const confirmCreate = async (): Promise<void> => {
      fireEvent.click(await screen.findByRole("button", { name: /^Create/ }));
    };

    it("should lock the options while it creates an enumeration", async () => {
      const c = renderStates();
      await openCreate(c);
      await confirmCreate();
      expect(c.queryByRole("button", { name: "Add entry" })).toBeNull();
      await created(c.value);
    });

    it("should not offer Create enumeration to a user who cannot create", async () => {
      const viewer = await new RoleClients(client).get("Viewer");
      const viewerWrapper = await createAsyncSynnaxWrapper({ client: viewer });
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
      await confirmCreate();
      const e = await created(c.value);
      expect(
        e.entries.map(({ name, value, color: ref }) => ({ name, value, ref })),
      ).toEqual([
        { name: "Off", value: 0, ref: undefined },
        { name: "On", value: 1, ref: { type: "fixed", value: OPTIONS[1].color } },
      ]);
    });

    it("should ask for the name of an unlinked node's new enumeration", async () => {
      const label = `Pump ${id.create()}`;
      const c = renderStates({ label: { label } });
      await openCreate(c);
      expect((await findInput("Enumeration name")).value).toBe(label);
      await confirmCreate();
      expect((await created(c.value)).name).toBe(label);
    });

    it("should start the name with the symbol for a node with no label", async () => {
      const c = renderStates();
      await openCreate(c);
      expect((await findInput("Enumeration name")).value).toBe("State indicator");
      await confirmCreate();
      expect((await created(c.value)).name).toBe("State indicator");
    });

    it("should show an error at the name field on a Create with no name", async () => {
      const c = renderStates();
      await openCreate(c);
      const field = await findInput("Enumeration name");
      fireEvent.change(field, { target: { value: " " } });
      await confirmCreate();
      expect(await screen.findByText("Name is required")).toBeTruthy();
      fireEvent.change(field, { target: { value: "Pump" } });
      expect(screen.queryByText("Name is required")).toBeNull();
      await act(async () => {});
      expect(c.value(path.enumeration)).toBeUndefined();
    });

    it("should close the modal with one Escape from the name field", async () => {
      const c = renderStates();
      await openCreate(c);
      const field = await findInput("Enumeration name");
      fireEvent.keyDown(field, { key: "Escape", code: "Escape" });
      fireEvent.keyUp(field, { key: "Escape", code: "Escape" });
      await waitFor(() =>
        expect(screen.queryByLabelText("Enumeration name")).toBeNull(),
      );
      expect(c.value(path.enumeration)).toBeUndefined();
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
      const field = await findInput("Enumeration name");
      expect(field.value).toBe("");
      expect(c.value(path.enumeration)).toBe(linked.key);
      const name = `Pump ${id.create()}`;
      fireEvent.change(field, { target: { value: name } });
      await confirmCreate();
      const e = await created(c.value, linked.key);
      expect(e.name).toBe(name);
      expect(e.entries).toEqual([]);
      await waitFor(() =>
        expect(document.activeElement).toBe(
          c.getByRole("button", { name: "Add entry" }),
        ),
      );
      expect(screen.queryByLabelText("Enumeration name")).toBeNull();
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
