// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, schematic, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { uniqueName } from "@synnaxlabs/x/testutil";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import type { PropsWithChildren, ReactElement } from "react";
import { expect } from "vitest";

import { Enumeration } from "@/feature/enumeration";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import {
  CaptureStatuses,
  clickAndSettle,
  createConsoleWrapper,
  HaulProvider,
  openContextMenu,
} from "@/testutil";

export const client = createTestClient();
// Reads what the Core broadcasts, without the edits that the cache of client shows at
// once. Its cache hears a change a moment after the write that made it returns.
export const core = createTestClient();

export const createEnumeration = async (
  entries: enumeration.Entry[] = [
    { key: "a", value: 0, name: uniqueName("a"), abnormal: false },
    { key: "b", value: 1, name: uniqueName("b"), abnormal: false },
    { key: "c", value: 2, name: uniqueName("c"), abnormal: false },
  ],
): Promise<enumeration.Enumeration> =>
  await client.enumerations.create({ name: uniqueName("enumeration"), entries });

/** Creates a schematic whose one select draws the enumeration. */
export const createSelectUser = async (
  e: enumeration.Key,
): Promise<schematic.Schematic> => {
  const project = await client.projects.create({ name: uniqueName("project") });
  return await client.schematics.create({
    project: project.key,
    name: uniqueName("schematic"),
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

export interface Rendered {
  /** Returns the status notifications that the Console shows now. */
  statuses: () => Status.NotificationSpec[];
}

export const renderEnumerations = async (as: Synnax = client): Promise<Rendered> => {
  const { wrapper: Console } = await createConsoleWrapper({ client: as });
  let statuses: Status.NotificationSpec[] = [];
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Tree.Provider items={{}}>
        <HaulProvider>{children}</HaulProvider>
        <Modals.Stack />
      </Tree.Provider>
    </Console>
  );
  await act(async () => {
    render(
      <>
        <Enumeration.Settings />
        <CaptureStatuses onStatuses={(next) => (statuses = next)} />
      </>,
      { wrapper },
    );
  });
  return { statuses: () => statuses };
};

export type Queries = ReturnType<typeof within>;

/**
 * Returns the queries of the dialog that holds the element. They stay fast while the
 * page shows each enumeration on the Core.
 */
export const dialogOf = (el: HTMLElement): Queries => {
  const dialog = screen.getAllByRole("dialog").find((d) => d.contains(el));
  if (dialog == null) throw new Error("expected a dialog");
  return within(dialog);
};

export const openDelete = async (name: string): Promise<Queries> => {
  await openContextMenu(name);
  await clickAndSettle("Delete");
  return dialogOf(await screen.findByText(/Are you sure you want to delete/));
};

export const pickReplacement = async (dialog: Queries, name: string): Promise<void> => {
  fireEvent.click(await dialog.findByRole("button", { name: /Replace with/ }));
  const search = await screen.findByPlaceholderText("Search enumerations...");
  fireEvent.change(search, { target: { value: name } });
  const dropdown = dialogOf(search);
  fireEvent.click(await dropdown.findByRole("option", { name: new RegExp(name) }));
};

/** Clicks a delete dialog button once the dialog knows what uses the enumeration. */
export const clickWhenEnabled = async (
  dialog: Queries,
  name: string,
): Promise<void> => {
  const button = await dialog.findByRole("button", { name });
  await waitFor(() => expect(button.getAttribute("aria-disabled")).toBeNull());
  await act(async () => {
    fireEvent.click(button);
  });
};

/** Waits for the delete dialog to close, which it does once the Core commits. */
export const waitForDialogToClose = async (): Promise<void> =>
  await waitFor(() =>
    expect(screen.queryByText(/Are you sure you want to delete/)).toBeNull(),
  );
