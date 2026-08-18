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

import { binary } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";
import { z } from "zod";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({
  engine: "web",
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

import { open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";

import { FS } from "@/platform/fs";
import { fakePickedFile, interceptFilePicker, renderWithConsole } from "@/testutil";

const openMock = vi.mocked(open);
const readFileMock = vi.mocked(readFile);

let dir: string;

beforeAll(async () => {
  dir = await mkdtemp(join(tmpdir(), "console-fs-spec-"));
});

afterAll(async () => {
  await rm(dir, { recursive: true, force: true });
});

describe("FS.InputFile", () => {
  beforeEach(() => {
    mocks.engine = "web";
    openMock.mockReset();
    readFileMock.mockClear();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should show the caller's file name without reading anything", async () => {
    const onChange = vi.fn();
    await renderWithConsole(
      <FS.InputFile
        value="table.csv"
        onChange={onChange}
        title="Select file"
        extension="csv"
      />,
    );
    expect(screen.getByText("table.csv")).toBeTruthy();
    expect(readFileMock).not.toHaveBeenCalled();
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should read a file picked in the browser", async () => {
    const picker = interceptFilePicker();
    const onChange = vi.fn();
    await renderWithConsole(
      <FS.InputFile value="" onChange={onChange} title="Select file" extension="txt" />,
    );
    fireEvent.click(screen.getByText("Select file"));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("notes.txt", "file body")]);
    await waitFor(() =>
      expect(onChange).toHaveBeenCalledWith("file body", "notes.txt"),
    );
  });

  it("should decode JSON contents against the provided schema", async () => {
    const picker = interceptFilePicker();
    const schema = z.object({ host: z.string(), port: z.number() });
    const onChange = vi.fn();
    await renderWithConsole(
      <FS.InputFile
        value=""
        onChange={onChange}
        title="Select file"
        extension="json"
        schema={schema}
        decoder={binary.JSON_CODEC}
      />,
    );
    fireEvent.click(screen.getByText("Select file"));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([
      fakePickedFile("config.json", JSON.stringify({ host: "localhost", port: 9090 })),
    ]);
    await waitFor(() =>
      expect(onChange).toHaveBeenCalledWith(
        { host: "localhost", port: 9090 },
        "config.json",
      ),
    );
  });

  describe("tauri engine", () => {
    beforeEach(() => {
      mocks.engine = "tauri";
    });

    it("should read the file behind the chosen path", async () => {
      const path = join(dir, "chosen.txt");
      await writeFile(path, "chosen body");
      openMock.mockResolvedValue(path);
      const onChange = vi.fn();
      await renderWithConsole(
        <FS.InputFile
          value=""
          onChange={onChange}
          title="Select file"
          extension="txt"
        />,
      );
      fireEvent.click(screen.getByText("Select file"));
      await waitFor(() =>
        expect(onChange).toHaveBeenCalledWith("chosen body", "chosen.txt"),
      );
    });

    it("should not call onChange when the chosen file does not exist", async () => {
      const path = join(dir, "missing.txt");
      openMock.mockResolvedValue(path);
      const onChange = vi.fn();
      await renderWithConsole(
        <FS.InputFile
          value=""
          onChange={onChange}
          title="Select file"
          extension="txt"
        />,
      );
      fireEvent.click(screen.getByText("Select file"));
      await waitFor(() => expect(readFileMock).toHaveBeenCalledWith(path));
      expect(onChange).not.toHaveBeenCalled();
    });
  });
});
