// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import type { Status } from "@synnaxlabs/lyra/status";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { ICD } from "@/feature/icd";
import { client, ENGINE_CSV } from "@/feature/icd/testutil";
import { Session } from "@/session";
import {
  createTestClientWithGrants,
  fakePickedFile,
  interceptFilePicker,
  resolveFocusedTab,
} from "@/testutil";

/** Opens the palette in a new project, capturing raised statuses. */
const openPalette = async () => {
  const project = await client.projects.create({ name: uniqueName("project") });
  const statuses: Status.NotificationSpec[] = [];
  const palette = await renderPalette({
    commands: ICD.COMMANDS,
    client,
    onStatuses: (next) => statuses.splice(0, statuses.length, ...next),
  });
  palette.store.dispatch(Session.Project.select(project.key));
  await palette.openCommandPalette();
  return { ...palette, statuses };
};

/** Picks file through the Import ICD command. */
const importFile = async (file: File) => {
  const picker = interceptFilePicker();
  const palette = await openPalette();
  await palette.selectCommand("Import ICD");
  await waitFor(() => picker.lastInput());
  picker.selectFiles([file]);
  return palette;
};

const retrieveFocusedICD = async (
  palette: Awaited<ReturnType<typeof openPalette>>,
): Promise<icd.ICD> => {
  const tab = await resolveFocusedTab(palette.store, client);
  if (tab.variant !== "resource") throw new Error("expected a resource tab");
  expect(tab.resource.type).toBe(icd.TYPE_RESOURCE_ID.type);
  return await client.icds.retrieve({ key: tab.resource.key });
};

describe("ICD commands", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should create an ICD and open it in a tab", async () => {
    const palette = await openPalette();
    await palette.selectCommand("Create ICD");
    expect(await retrieveFocusedICD(palette)).toMatchObject({
      name: "ICD",
      messages: [],
    });
  });

  it("should create an ICD named after an imported file and open it in a tab", async () => {
    const name = uniqueName("engine");
    const palette = await importFile(fakePickedFile(`${name}.csv`, ENGINE_CSV));
    const created = await retrieveFocusedICD(palette);
    expect(created.name).toBe(name);
    expect(created.messages.map((m) => m.name)).toEqual(["Engine"]);
  });

  it("should hide Import ICD from a user who cannot update ICDs", async () => {
    const creator = await createTestClientWithGrants(client, {
      retrieve: [icd.TYPE_RESOURCE_ID],
      create: [icd.TYPE_RESOURCE_ID],
    });
    const { openCommandPalette } = await renderPalette({
      commands: ICD.COMMANDS,
      client: creator,
    });
    await openCommandPalette();
    await screen.findByText("Create ICD");
    expect(screen.queryByText("Import ICD")).toBeNull();
  });

  it("should list the accepted columns of a file with an unknown column", async () => {
    const name = uniqueName("plant");
    const { statuses } = await importFile(
      fakePickedFile(`${name}.csv`, "Name,ID\nrpm,1\n"),
    );
    await waitFor(() => expect(statuses).toHaveLength(1));
    expect(statuses[0]).toMatchObject({
      variant: "error",
      message: "Failed to import ICD",
      description:
        'data: row 1: unknown column "Name": expected message, id, extended, ' +
        "length, field, start_bit, bit_length, byte_order, kind, scale, offset, " +
        "units, min, or max",
    });
    const named = (await client.icds.retrieve({})).filter((ic) => ic.name === name);
    expect(named).toEqual([]);
  });
});
