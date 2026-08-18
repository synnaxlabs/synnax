// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/pluto";
import { act, createEvent, fireEvent, screen } from "@testing-library/react";
import { type DragEvent, type ReactElement } from "react";

import { type FileImporterContext } from "@/platform/import/import";

const START_FILE_DRAG = "start file drag";

/**
 * Renders a button that puts an OS file haul in flight, the way the window Chrome
 * does on dragover in production (app/window/Window.tsx). Render it beside the drop
 * target, inside the same Haul provider.
 */
export const FileDragSource = (): ReactElement => {
  const { startDrag } = Haul.useDrag({ type: "os-file", key: "os-file" });
  return <button onClick={() => startDrag([Haul.FILE])}>{START_FILE_DRAG}</button>;
};

/** Starts the file haul a rendered {@link FileDragSource} provides. */
export const startFileDrag = (): void => {
  act(() => {
    fireEvent.click(screen.getByText(START_FILE_DRAG));
  });
};

/**
 * Builds a FileImporterContext with no connected cluster. Merge overrides over it for
 * spec-specific fields.
 */
export const createFileImporterContext = (
  overrides: Partial<FileImporterContext> = {},
): FileImporterContext => ({
  client: null,
  parent: project.ontologyID("project-1"),
  fileName: "test.json",
  ...overrides,
});

/** Builds a JSON file the drop path accepts. */
export const createJSONFile = (name: string, contents: unknown): File =>
  new File([JSON.stringify(contents)], name, { type: "application/json" });

/**
 * Builds the FileSystemFileEntry surface the drop path reads. jsdom cannot construct
 * one, so the sanctioned casts to the DOM types live here. Pass `until` to hold the
 * read open, which orders one entry of a drop against the others.
 */
export const fakeFileEntry = (file: File, until?: Promise<void>): FileSystemEntry =>
  ({
    isFile: true,
    isDirectory: false,
    name: file.name,
    file: (resolve: (file: File) => void) => {
      if (until == null) resolve(file);
      else void until.then(() => resolve(file));
    },
  }) as unknown as FileSystemEntry;

/**
 * Builds the FileSystemDirectoryEntry surface the drop path reads. A File child reads
 * as a file entry; a FileSystemEntry child passes through, so directories nest. Pass
 * batchSize to hand out children over several readEntries calls, the way Chrome caps
 * each call at 100 entries.
 */
export const fakeDirectoryEntry = (
  name: string,
  children: (File | FileSystemEntry)[],
  batchSize: number = children.length,
): FileSystemEntry =>
  ({
    isFile: false,
    isDirectory: true,
    name,
    createReader: () => {
      let offset = 0;
      return {
        readEntries: (resolve: (entries: FileSystemEntry[]) => void) => {
          const batch = children.slice(offset, offset + batchSize);
          offset += batchSize;
          resolve(
            batch.map((child) =>
              child instanceof File ? fakeFileEntry(child) : child,
            ),
          );
        },
      };
    },
  }) as unknown as FileSystemEntry;

/**
 * Builds the DataTransfer surface the drop path reads, carrying the given entries. A
 * null entry becomes a non-file item, which the drop path skips.
 */
export const fakeDataTransfer = (entries: (FileSystemEntry | null)[]): DataTransfer =>
  ({
    items: entries.map((entry) => ({
      kind: entry == null ? "string" : "file",
      webkitGetAsEntry: () => entry,
    })),
  }) as unknown as DataTransfer;

/** Builds the drag event the mosaic hands to a file drop handler. */
export const fakeFileDropEvent = (entries: (FileSystemEntry | null)[]): DragEvent =>
  ({ dataTransfer: fakeDataTransfer(entries) }) as unknown as DragEvent;

/**
 * Fires an OS file drop carrying the given entries at the target's center. jsdom has no
 * DragEvent, so the cursor and the transfer are defined on the event testing-library
 * falls back to.
 */
export const fireFileDrop = (
  target: HTMLElement,
  entries: (FileSystemEntry | null)[],
): void => {
  const event = createEvent.drop(target);
  Object.defineProperties(event, {
    clientX: { value: 50 },
    clientY: { value: 50 },
    dataTransfer: { value: fakeDataTransfer(entries) },
  });
  fireEvent(target, event);
};
