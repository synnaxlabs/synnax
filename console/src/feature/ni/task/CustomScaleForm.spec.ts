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

vi.mock("@tauri-apps/api/path", () => ({ sep: vi.fn(() => "/") }));
vi.mock("@tauri-apps/plugin-dialog", () => ({ open: vi.fn() }));
// The Tauri fs plugin is an unmockable IPC seam in jsdom; route it to the real node
// filesystem so the specs exercise genuine file reads.
vi.mock("@tauri-apps/plugin-fs", async () => {
  const { readFile } = await import("node:fs/promises");
  return {
    readDir: vi.fn(),
    readFile: vi.fn(async (path: string) => new Uint8Array(await readFile(path))),
    readTextFile: vi.fn(),
  };
});

import { type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { open } from "@tauri-apps/plugin-dialog";

import { NI } from "@/feature/ni";
import { renderNITaskForm } from "@/feature/ni/task/testutil";
import {
  commitFieldInput,
  findDialogTriggerByText,
  selectFromDropdown,
} from "@/platform/task/testutil";
import {
  fakePickedFile,
  getIconButton,
  getInputTable,
  interceptFilePicker,
} from "@/testutil";

const openMock = vi.mocked(open);

const client = createTestClient();

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<NI.Task.AnalogReadSchemas> = {
  name: "NI analog read task",
  type: NI.Task.ANALOG_READ_TYPE,
  config: NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
};

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
  ...(NI.Task.createAIChannel("ai_voltage") as VoltageChannel),
  key: id.create(),
  port: 0,
  device: "placeholder_device",
  customScale,
});

const renderWithScale = async (customScale: NI.Task.Scale) => {
  const config: task.Payload<NI.Task.AnalogReadSchemas>["config"] = {
    ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
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
    await renderWithScale(NI.Task.createScale("none"));
    await screen.findByText("Custom scaling");
    expect(screen.queryByText("Slope")).toBeNull();
    await selectFromDropdown("None", "Linear");
    await waitFor(() => expect(screen.getByText("Slope")).toBeTruthy());
    expect(screen.getByText("Y-Intercept")).toBeTruthy();
    await selectFromDropdown("Linear", "Map");
    await waitFor(() => expect(screen.getByText("Pre-scaled min")).toBeTruthy());
    expect(screen.queryByText("Slope")).toBeNull();
  });

  it("should populate the column selects from a loaded table CSV", async () => {
    const csvPath = join(dir, "table.csv");
    await writeFile(csvPath, "raw_col,scaled_col\n1,10\n2,20\n3,30\n");
    openMock.mockResolvedValue(csvPath);
    await renderWithScale(NI.Task.createScale("table"));
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
    await renderWithScale(NI.Task.createScale("table"));
    await screen.findByText("Table CSV");
    fireEvent.click(screen.getByText("Select file"));
    await findDialogTriggerByText("col_b");
    await selectFromDropdown("col_b", "col_c");
    await findDialogTriggerByText("col_c");
  });

  it("should load a table CSV in the browser runtime", async () => {
    mocks.engine = "web";
    openMock.mockClear();
    const picker = interceptFilePicker();
    await renderWithScale(NI.Task.createScale("table"));
    await screen.findByText("Table CSV");
    fireEvent.click(screen.getByText("Select file"));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([
      fakePickedFile("browser.csv", "web_raw,web_scaled\n1,10\n2,20\n"),
    ]);
    await findDialogTriggerByText("web_raw");
    await findDialogTriggerByText("web_scaled");
    expect(screen.getByText(/browser\.csv/)).toBeTruthy();
    expect(openMock).not.toHaveBeenCalled();
  });

  describe("manual entry", () => {
    it("should append rows with an increasing pre-scaled value", async () => {
      await renderWithScale(NI.Task.createScale("table"));
      await screen.findByText("Values");
      expect(getInputTable("Values").rows).toHaveLength(0);
      fireEvent.click(getInputTable("Values").add);
      await waitFor(() => expect(getInputTable("Values").rows).toHaveLength(1));
      fireEvent.click(getInputTable("Values").add);
      await waitFor(() => expect(getInputTable("Values").rows).toHaveLength(2));
      const table = getInputTable("Values");
      expect(table.cell(0, 0).value).toBe("0");
      expect(table.cell(1, 0).value).toBe("1");
    });

    it("should keep an edited pair in the table", async () => {
      await renderWithScale({
        ...(NI.Task.createScale("table") as Extract<NI.Task.Scale, { type: "table" }>),
        preScaledVals: [1, 2],
        scaledVals: [10, 20],
      });
      await screen.findByText("Values");
      commitFieldInput(getInputTable("Values").cell(1, 1), "25");
      await waitFor(() => expect(getInputTable("Values").cell(1, 1).value).toBe("25"));
      expect(getInputTable("Values").cell(0, 1).value).toBe("10");
    });

    it("should remove the row whose button is clicked", async () => {
      await renderWithScale({
        ...(NI.Task.createScale("table") as Extract<NI.Task.Scale, { type: "table" }>),
        preScaledVals: [1, 2, 3],
        scaledVals: [10, 20, 30],
      });
      await screen.findByText("Values");
      fireEvent.click(getIconButton(getInputTable("Values").rows[1], "close"));
      await waitFor(() => expect(getInputTable("Values").rows).toHaveLength(2));
      const table = getInputTable("Values");
      expect(table.cell(0, 0).value).toBe("1");
      expect(table.cell(1, 0).value).toBe("3");
    });

    it("should fill the table from a pasted block", async () => {
      await renderWithScale(NI.Task.createScale("table"));
      await screen.findByText("Values");
      fireEvent.click(getInputTable("Values").add);
      await waitFor(() => expect(getInputTable("Values").rows).toHaveLength(1));
      const cell = getInputTable("Values").cell(0, 0);
      fireEvent.focus(cell);
      fireEvent.paste(cell, {
        clipboardData: { getData: () => "1\t10\n2\t20\n3\t30" },
      });
      await waitFor(() => expect(getInputTable("Values").rows).toHaveLength(3));
      const table = getInputTable("Values");
      expect(table.cell(2, 0).value).toBe("3");
      expect(table.cell(2, 1).value).toBe("30");
    });

    it("should reject a CSV whose columns hold a different number of values", async () => {
      const csvPath = join(dir, "ragged.csv");
      await writeFile(csvPath, "raw_col,scaled_col\n1,10\n2,\n3,30\n");
      openMock.mockResolvedValue(csvPath);
      await renderWithScale(NI.Task.createScale("table"));
      await screen.findByText("Table CSV");
      fireEvent.click(screen.getByText("Select file"));
      await screen.findByText(
        "Pre-scaled 3 values and scaled 2 values must be the same length",
      );
      expect(getInputTable("Values").rows).toHaveLength(0);
    });

    it("should show the values loaded from a CSV", async () => {
      const csvPath = join(dir, "table3.csv");
      await writeFile(csvPath, "raw_col,scaled_col\n1,10\n2,20\n3,30\n");
      openMock.mockResolvedValue(csvPath);
      await renderWithScale(NI.Task.createScale("table"));
      await screen.findByText("Table CSV");
      fireEvent.click(screen.getByText("Select file"));
      await waitFor(() => expect(getInputTable("Values").rows).toHaveLength(3));
      const table = getInputTable("Values");
      expect(table.cell(0, 0).value).toBe("1");
      expect(table.cell(2, 1).value).toBe("30");
    });
  });
});
