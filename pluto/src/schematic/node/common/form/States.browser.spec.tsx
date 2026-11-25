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
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { deep, id } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, beforeEach, describe, expect, it, type Mock, vi } from "vitest";
import { page } from "vitest/browser";
import type { z } from "zod";

import { Enumeration } from "@/enumeration";
import { Form } from "@/schematic/node/common/form";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.StateIndicatorNodeConfig>();

const client = createTestClient();

const findInput = async (label: string): Promise<HTMLInputElement> =>
  await screen.findByLabelText<HTMLInputElement>(label);

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
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createWrapper();
  });

  const renderStates = (
    config: Partial<z.input<typeof schematic.stateIndicatorNodeConfigZ>> = {},
    as: FC<PropsWithChildren> = wrapper,
  ) =>
    renderForm(
      <Form.States
        enumeration={path.enumeration}
        label={path.label.label}
        symbolName="State indicator"
      />,
      schematic.stateIndicatorNodeConfigZ.parse({
        variant: "state_indicator",
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

    // A listed option can render again between a find and a later click, so the click
    // finds the option as it acts.
    const pick = async (
      c: ReturnType<typeof renderStates>,
      e: enumeration.Enumeration,
    ): Promise<void> => {
      await openPicker(c, e.name);
      await page.getByRole("option", { name: e.name, exact: true }).click();
    };

    it("should not offer Create enumeration to a user who cannot create", async () => {
      const viewer = await new RoleClients(client).get("Viewer");
      const viewerWrapper = await createWrapper(viewer);
      // A render as a user who is not an admin suspends at first.
      let c: ReturnType<typeof renderStates> | undefined;
      await act(async () => {
        c = renderStates({}, viewerWrapper);
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

    it("should create an empty enumeration for a node that names none", async () => {
      const c = renderStates();
      await openCreate(c);
      const e = await created(c.value);
      expect(prompt).toHaveBeenCalledWith(expect.objectContaining({ entries: [] }));
      expect(e.entries).toEqual([]);
      await waitFor(() =>
        expect(document.activeElement).toBe(
          c.getByRole("button", { name: "Add entry" }),
        ),
      );
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

    it("should name no enumeration when the user cancels the name", async () => {
      prompt.mockResolvedValueOnce(null);
      const c = renderStates();
      await openCreate(c);
      await act(async () => {});
      expect(prompt).toHaveBeenCalledOnce();
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
      await pick(c, linked);
      await waitFor(() => expect(c.value(path.enumeration)).toBe(linked.key));
      await pick(c, e);
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

    it("should show no entries on No enumeration", async () => {
      const e = await client.enumerations.create({
        name: id.create(),
        entries: [{ key: id.create(), name: "Primed", value: 0, abnormal: false }],
      });
      const c = renderStates({ enumeration: e.key });
      await findInput("Name 1");
      fireEvent.click(await c.findByRole("button", { name: /^Enumeration/ }));
      fireEvent.click(await c.findByText("No enumeration"));
      await waitFor(() => expect(c.value(path.enumeration)).toBeUndefined());
      expect(c.queryByLabelText("Name 1")).toBeNull();
    });

    it("should span the picker across the columns of the entries", async () => {
      const e = await client.enumerations.create({
        name: id.create(),
        entries: [{ key: id.create(), name: "Primed", value: 0, abnormal: false }],
      });
      const c = renderStates({ enumeration: e.key });
      await findInput("Name 1");
      const picker = c
        .getByRole("button", { name: /^Enumeration/ })
        .getBoundingClientRect();
      const columns = c
        .getAllByRole("columnheader")
        .filter((th) => th.textContent !== "");
      const first = columns[0].getBoundingClientRect();
      const last = columns[columns.length - 1].getBoundingClientRect();
      expect(picker.left).toBeCloseTo(first.left, 0);
      expect(picker.right).toBeCloseTo(last.right, 0);
    });

    it("should keep the picker as wide when the node names no enumeration", async () => {
      const e = await client.enumerations.create({
        name: id.create(),
        entries: [{ key: id.create(), name: "Primed", value: 0, abnormal: false }],
      });
      const c = renderStates({ enumeration: e.key });
      await findInput("Name 1");
      const picker = (): HTMLElement => c.getByRole("button", { name: /^Enumeration/ });
      const linked = picker().getBoundingClientRect().width;
      fireEvent.click(picker());
      fireEvent.click(await c.findByText("No enumeration"));
      await waitFor(() => expect(c.queryByLabelText("Name 1")).toBeNull());
      expect(picker().getBoundingClientRect().width).toBeCloseTo(linked, 0);
    });
  });
});
