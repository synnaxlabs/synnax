// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({
  engine: "tauri",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/plugin-dialog", () => ({ open: vi.fn() }));
// The Tauri fs plugin is an unmockable IPC seam in jsdom; route it to the real node
// filesystem so the specs exercise genuine file reads.
vi.mock("@tauri-apps/plugin-fs", async () => {
  const { readFile } = await import("node:fs/promises");
  return {
    readFile: vi.fn(async (path: string) => new Uint8Array(await readFile(path))),
  };
});

import { type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { open } from "@tauri-apps/plugin-dialog";

import { NI } from "@/feature/ni";
import { renderNITaskForm } from "@/feature/ni/task/testutil";
import { findDialogTriggerByText, selectFromDropdown } from "@/platform/task/testutil";

const openMock = vi.mocked(open);

const client = createTestClient();

// Draft creates mint their own key; the zero payload's empty key must not be sent.
const { key: _key, ...ZERO_DRAFT } = NI.Task.ZERO_ANALOG_READ_PAYLOAD;

let dir: string;

beforeAll(async () => {
  dir = await mkdtemp(join(tmpdir(), "console-ni-scale-spec-"));
});

afterAll(async () => {
  await rm(dir, { recursive: true, force: true });
});

beforeEach(() => {
  mocks.engine = "tauri";
});

type VoltageChannel = Extract<NI.Task.AIChannel, { type: "ai_voltage" }>;

const createChannel = (customScale: NI.Task.Scale): VoltageChannel => ({
  ...(NI.Task.ZERO_AI_CHANNELS.ai_voltage as VoltageChannel),
  key: id.create(),
  port: 0,
  device: "placeholder_device",
  customScale,
});

const renderWithScale = async (customScale: NI.Task.Scale) => {
  const config: task.Payload<NI.Task.AnalogReadSchemas>["config"] = {
    ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
    channels: [createChannel(customScale)],
  };
  const draft = await client.tasks.create(
    { ...ZERO_DRAFT, config },
    NI.Task.ANALOG_READ_SCHEMAS,
  );
  return await renderNITaskForm(NI.Task.AnalogRead, {
    client,
    taskKey: draft.key,
  });
};

describe("CustomScaleForm", () => {
  it("should swap the scale fields when a different scale type is selected", async () => {
    await renderWithScale(NI.Task.ZERO_SCALES.none);
    await screen.findByText("Custom Scaling");
    expect(screen.queryByText("Slope")).toBeNull();
    await selectFromDropdown("None", "Linear");
    await waitFor(() => expect(screen.getByText("Slope")).toBeTruthy());
    expect(screen.getByText("Y-Intercept")).toBeTruthy();
    await selectFromDropdown("Linear", "Map");
    await waitFor(() => expect(screen.getByText("Pre-Scaled Min")).toBeTruthy());
    expect(screen.queryByText("Slope")).toBeNull();
  });

  it("should populate the column selects from a loaded table CSV", async () => {
    const csvPath = join(dir, "table.csv");
    await writeFile(csvPath, "raw_col,scaled_col\n1,10\n2,20\n3,30\n");
    openMock.mockResolvedValue(csvPath);
    await renderWithScale(NI.Task.ZERO_SCALES.table);
    await screen.findByText("Table CSV");
    fireEvent.click(screen.getByText("Select file"));
    await findDialogTriggerByText("raw_col");
    await findDialogTriggerByText("scaled_col");
    expect(screen.getByText(/table\.csv/)).toBeTruthy();
  });

  it("should let the scaled column be reassigned after a load", async () => {
    const csvPath = join(dir, "table2.csv");
    await writeFile(csvPath, "col_a,col_b,col_c\n1,10,100\n2,20,200\n");
    openMock.mockResolvedValue(csvPath);
    await renderWithScale(NI.Task.ZERO_SCALES.table);
    await screen.findByText("Table CSV");
    fireEvent.click(screen.getByText("Select file"));
    await findDialogTriggerByText("col_b");
    await selectFromDropdown("col_b", "col_c");
    await findDialogTriggerByText("col_c");
  });
});
