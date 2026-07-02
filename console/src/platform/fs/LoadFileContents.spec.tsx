// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({
  engine: "web",
}));

vi.mock("@/session/runtime/runtime", () => ({
  get ENGINE() {
    return mocks.engine;
  },
  Drift: class {},
}));

vi.mock("@tauri-apps/plugin-dialog", () => ({ open: vi.fn() }));
vi.mock("@tauri-apps/plugin-fs", () => ({ readFile: vi.fn() }));

import { open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";

import { FS } from "@/platform/fs";
import { renderWithConsole } from "@/testutil";

const openMock = vi.mocked(open);
const readFileMock = vi.mocked(readFile);

const bytes = (s: string): Uint8Array<ArrayBuffer> =>
  Uint8Array.from(new TextEncoder().encode(s));

describe("FS.InputFilePath", () => {
  beforeEach(() => {
    mocks.engine = "web";
    openMock.mockReset();
    readFileMock.mockReset();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should show a placeholder when no file is selected", async () => {
    await renderWithConsole(
      <FS.InputFilePath value={undefined as unknown as string} onChange={vi.fn()} />,
    );
    expect(screen.getByText("No file selected")).toBeTruthy();
  });

  it("should show the selected path", async () => {
    await renderWithConsole(
      <FS.InputFilePath value="/tmp/config.json" onChange={vi.fn()} />,
    );
    expect(screen.getByText("/tmp/config.json")).toBeTruthy();
  });

  it("should surface an error and not call onChange when clicked in the browser", async () => {
    const onChange = vi.fn();
    await renderWithConsole(<FS.InputFilePath value="" onChange={onChange} />);
    fireEvent.click(screen.getByText("Select file"));
    await waitFor(() => expect(openMock).not.toHaveBeenCalled());
    expect(onChange).not.toHaveBeenCalled();
  });

  describe("tauri engine", () => {
    beforeEach(() => {
      mocks.engine = "tauri";
    });

    it("should open a dialog and report the chosen path", async () => {
      openMock.mockResolvedValue("/tmp/picked.json");
      const onChange = vi.fn();
      await renderWithConsole(
        <FS.InputFilePath
          value=""
          onChange={onChange}
          filters={[{ name: "JSON", extensions: ["json"] }]}
        />,
      );
      fireEvent.click(screen.getByText("Select file"));
      await waitFor(() => expect(onChange).toHaveBeenCalledWith("/tmp/picked.json"));
      expect(openMock).toHaveBeenCalledWith({
        directory: false,
        filters: [{ name: "JSON", extensions: ["json"] }],
      });
    });

    it("should not call onChange when the dialog is cancelled", async () => {
      openMock.mockResolvedValue(null);
      const onChange = vi.fn();
      await renderWithConsole(<FS.InputFilePath value="" onChange={onChange} />);
      fireEvent.click(screen.getByText("Select file"));
      await waitFor(() => expect(openMock).toHaveBeenCalled());
      expect(onChange).not.toHaveBeenCalled();
    });
  });
});

describe("FS.InputFileContents", () => {
  beforeEach(() => {
    mocks.engine = "tauri";
    openMock.mockReset();
    readFileMock.mockReset();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should read and decode the file at the initial path", async () => {
    readFileMock.mockResolvedValue(bytes("file body"));
    const onChange = vi.fn();
    await renderWithConsole(
      <FS.InputFileContents onChange={onChange} initialPath="/tmp/seed.txt" />,
    );
    await waitFor(() =>
      expect(onChange).toHaveBeenCalledWith("file body", "/tmp/seed.txt"),
    );
    expect(readFileMock).toHaveBeenCalledWith("/tmp/seed.txt");
    expect(screen.getByText("/tmp/seed.txt")).toBeTruthy();
  });

  it("should read the file when the path is chosen via the dialog", async () => {
    openMock.mockResolvedValue("/tmp/chosen.txt");
    readFileMock.mockResolvedValue(bytes("chosen body"));
    const onChange = vi.fn();
    await renderWithConsole(<FS.InputFileContents onChange={onChange} />);
    fireEvent.click(screen.getByText("Select file"));
    await waitFor(() =>
      expect(onChange).toHaveBeenCalledWith("chosen body", "/tmp/chosen.txt"),
    );
  });

  it("should not call onChange when the file read returns null", async () => {
    readFileMock.mockResolvedValue(null as unknown as Uint8Array<ArrayBuffer>);
    const onChange = vi.fn();
    await renderWithConsole(
      <FS.InputFileContents onChange={onChange} initialPath="/tmp/empty.txt" />,
    );
    await waitFor(() => expect(readFileMock).toHaveBeenCalledWith("/tmp/empty.txt"));
    expect(onChange).not.toHaveBeenCalled();
  });
});
