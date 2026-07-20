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
import { Icon } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
} from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Panel } from "@/platform/panel";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { awaitTextEditingElement, commitTextEdit, renderWithConsole } from "@/testutil";

const client = createTestClient();

describe("Panel tab", () => {
  describe("useTab", () => {
    it("resolves the renderer registered for the active tab's type", async () => {
      const tabKey = uuid.create();
      const existing = await createServerPanel(client, {
        variant: "leaf",
        tabs: [{ variant: "view", key: tabKey, type: "custom_view", args: {} }],
      });
      const { wrapper: Base } = await createPanelWrapper({
        client,
        panelKey: existing.key,
        tabKey,
      });
      await primePanel(Base, existing.key);
      const renderer: Panel.Tab = {
        Content: () => <>content</>,
        Name: () => <>name</>,
      };
      const renderers: Panel.Tabs = { custom_view: renderer };
      const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Base>
          <Panel.RendererContext value={renderers}>{children}</Panel.RendererContext>
        </Base>
      );
      const { result } = renderHook(() => Panel.useTab(), { wrapper: Wrapper });
      await waitFor(() => expect(result.current).toBe(renderer));
    });

    it("throws when no renderer is registered for the active tab's type", async () => {
      const tabKey = uuid.create();
      const existing = await createServerPanel(client, {
        variant: "leaf",
        tabs: [{ variant: "view", key: tabKey, type: "unregistered_view", args: {} }],
      });
      const { wrapper: Base } = await createPanelWrapper({
        client,
        panelKey: existing.key,
        tabKey,
      });
      await primePanel(Base, existing.key);
      const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Base>
          <Panel.RendererContext value={{}}>{children}</Panel.RendererContext>
        </Base>
      );
      expect(() => renderHook(() => Panel.useTab(), { wrapper: Wrapper })).toThrow(
        "no renderer for tab type unregistered_view",
      );
    });
  });

  describe("createStaticTabName", () => {
    it("renders a component displaying the provided name", async () => {
      const Name = Panel.createStaticTabName({
        name: "Static Tab Name",
        icon: <Icon.Schematic />,
      });
      await renderWithConsole(<Name />);
      expect(screen.getByText("Static Tab Name")).toBeTruthy();
    });
  });

  describe("createEditableTabName", () => {
    const renderName = async () => {
      const resourceKey = uuid.create();
      const tabKey = uuid.create();
      const existing = await createServerPanel(client, {
        variant: "leaf",
        tabs: [
          {
            variant: "resource",
            key: tabKey,
            resource: schematic.ontologyID(resourceKey),
          },
        ],
      });
      const { wrapper: Base } = await createPanelWrapper({
        client,
        panelKey: existing.key,
        tabKey,
      });
      await primePanel(Base, existing.key);
      const ensureRetrieved = vi.fn();
      const rename = vi.fn();
      const service: Panel.EditableTabNameService = {
        useEnsureRetrieved: ensureRetrieved,
        useSelectName: () => "Resolved Name",
        useRename: () => ({ update: rename }),
      };
      const Name = Panel.createEditableTabName(service, <Icon.Schematic />);
      render(<Name />, { wrapper: Base });
      await waitFor(() => expect(screen.getByText("Resolved Name")).toBeTruthy());
      return { resourceKey, ensureRetrieved, rename };
    };

    it("renders the injected service's name for the tab's resource", async () => {
      const { resourceKey, ensureRetrieved } = await renderName();
      expect(ensureRetrieved).toHaveBeenCalledWith({ key: resourceKey });
    });

    it("commits a rename through the injected service", async () => {
      const { resourceKey, rename } = await renderName();
      fireEvent.doubleClick(screen.getByText("Resolved Name"));
      const editing = await awaitTextEditingElement();
      act(() => commitTextEdit(editing, "Renamed Tab"));
      await waitFor(() =>
        expect(rename).toHaveBeenCalledWith({ key: resourceKey, name: "Renamed Tab" }),
      );
    });
  });
});
