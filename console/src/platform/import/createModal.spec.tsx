// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul, type Status } from "@synnaxlabs/pluto";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { unzipSync, zipSync } from "fflate";
import { type ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  createJSONFile,
  fakeDirectoryEntry,
  fakeFileEntry,
  FileDragSource,
  fireFileDrop,
  startFileDrag,
} from "@/platform/fs/testutil";
import { Import } from "@/platform/import";
import { Modals } from "@/platform/modals";
import {
  CaptureStatuses,
  fakePickedFile,
  interceptFilePicker,
  renderWithConsole,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
});

const ZONE_TEXT = "Drop a .zip or folder here";

const importFn =
  vi.fn<(bundle: Uint8Array<ArrayBuffer>, fileName: string) => Promise<string>>();

const useTestImport = Import.createModal({
  header: "Widget.Import",
  resourceName: "widget",
  useOnImport: () => importFn,
});

const Opener = (): ReactElement => {
  const open = useTestImport();
  return <button onClick={() => open()}>open import</button>;
};
Opener.displayName = "Opener";

const renderModal = async () => {
  importFn.mockReset();
  const statuses: Status.NotificationSpec[] = [];
  await renderWithConsole(
    <Haul.Provider>
      <FileDragSource />
      <Opener />
      <Modals.Stack />
      <CaptureStatuses
        onStatuses={(next) => statuses.splice(0, statuses.length, ...next)}
      />
    </Haul.Provider>,
  );
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "open import" }));
  });
  await screen.findByText(ZONE_TEXT);
  return { statuses };
};

describe("Import.createModal", () => {
  it("zips a dropped folder and imports it under the folder's name", async () => {
    const { statuses } = await renderModal();
    importFn.mockResolvedValue("My Widget");
    startFileDrag();
    fireFileDrop(screen.getByText(ZONE_TEXT), [
      fakeDirectoryEntry("my-widget", [createJSONFile("a.json", { type: "log" })]),
    ]);
    await waitFor(() => expect(importFn).toHaveBeenCalledTimes(1));
    expect(importFn.mock.calls[0][1]).toBe("my-widget");
    const entries = unzipSync(importFn.mock.calls[0][0]);
    expect(Object.keys(entries)).toEqual(["a.json"]);
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain('Imported widget "My Widget"'),
    );
    await waitFor(() => expect(screen.queryByText(ZONE_TEXT)).toBeNull());
  });

  it("hands a dropped .zip to the importer untouched", async () => {
    await renderModal();
    importFn.mockResolvedValue("Zipped");
    const bytes = zipSync({ "manifest.json": new TextEncoder().encode("{}") });
    startFileDrag();
    fireFileDrop(screen.getByText(ZONE_TEXT), [
      fakeFileEntry(new File([bytes], "My Widget.zip")),
    ]);
    await waitFor(() => expect(importFn).toHaveBeenCalledTimes(1));
    expect(importFn.mock.calls[0][0]).toEqual(bytes);
    expect(importFn.mock.calls[0][1]).toBe("My Widget.zip");
  });

  it("reports a dropped file that is not a .zip", async () => {
    const { statuses } = await renderModal();
    startFileDrag();
    fireFileDrop(screen.getByText(ZONE_TEXT), [
      fakeFileEntry(new File(["junk"], "notes.txt")),
    ]);
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to import widget"),
    );
    expect(importFn).not.toHaveBeenCalled();
    expect(screen.getByText(ZONE_TEXT)).toBeTruthy();
  });

  it("rejects a drop carrying more than one entry", async () => {
    const { statuses } = await renderModal();
    startFileDrag();
    fireFileDrop(screen.getByText(ZONE_TEXT), [
      fakeFileEntry(new File(["{}"], "a.zip")),
      fakeFileEntry(new File(["{}"], "b.zip")),
    ]);
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to import widget"),
    );
    expect(importFn).not.toHaveBeenCalled();
  });

  it("browses for a .zip when the zone is clicked", async () => {
    const picker = interceptFilePicker();
    const { statuses } = await renderModal();
    importFn.mockResolvedValue("Picked");
    const bytes = new Uint8Array(
      zipSync({ "manifest.json": new TextEncoder().encode("{}") }),
    );
    fireEvent.click(screen.getByText(ZONE_TEXT));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    expect(picker.lastInput().accept).toBe(".zip");
    picker.selectFiles([fakePickedFile("picked.zip", bytes)]);
    await waitFor(() => expect(importFn).toHaveBeenCalledTimes(1));
    expect(importFn.mock.calls[0][0]).toEqual(bytes);
    expect(importFn.mock.calls[0][1]).toBe("picked.zip");
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain('Imported widget "Picked"'),
    );
  });

  it("imports a folder picked with the folder button", async () => {
    const picker = interceptFilePicker();
    await renderModal();
    importFn.mockResolvedValue("Foldered");
    fireEvent.click(screen.getByRole("button", { name: "Select folder" }));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    expect(picker.lastInput().webkitdirectory).toBe(true);
    picker.selectFiles([fakePickedFile("a.json", "{}", "my-widget/a.json")]);
    await waitFor(() => expect(importFn).toHaveBeenCalledTimes(1));
    expect(importFn.mock.calls[0][1]).toBe("my-widget");
    const entries = unzipSync(importFn.mock.calls[0][0]);
    expect(Object.keys(entries)).toEqual(["a.json"]);
  });

  it("reports a failed import and keeps the modal open", async () => {
    const { statuses } = await renderModal();
    importFn.mockRejectedValue(new Error("boom"));
    const bytes = zipSync({ "manifest.json": new TextEncoder().encode("{}") });
    startFileDrag();
    fireFileDrop(screen.getByText(ZONE_TEXT), [
      fakeFileEntry(new File([bytes], "broken.zip")),
    ]);
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to import widget"),
    );
    expect(screen.getByText(ZONE_TEXT)).toBeTruthy();
  });
});
