// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createEvent, fireEvent } from "@testing-library/react";
import { type DragEvent } from "react";

import { type FileIngesterContext } from "@/platform/import/ingester";

/**
 * Builds a FileIngesterContext with no connected cluster. Merge overrides over it for
 * spec-specific fields.
 */
export const createFileIngesterContext = (
  overrides: Partial<FileIngesterContext> = {},
): FileIngesterContext => ({
  name: "test",
  client: null,
  projectKey: "project-1",
  fileName: "test.json",
  ...overrides,
});

/** Builds a JSON file the drop path accepts. */
export const createJSONFile = (name: string, contents: unknown): File =>
  new File([JSON.stringify(contents)], name, { type: "application/json" });

/**
 * Builds the FileSystemFileEntry surface the drop path reads. jsdom cannot construct
 * one, so the sanctioned casts to the DOM types live here.
 */
export const fakeFileEntry = (file: File): FileSystemEntry =>
  ({
    isFile: true,
    isDirectory: false,
    name: file.name,
    file: (resolve: (file: File) => void) => resolve(file),
  }) as unknown as FileSystemEntry;

/** Builds the FileSystemDirectoryEntry surface the drop path reads. */
export const fakeDirectoryEntry = (name: string, files: File[]): FileSystemEntry => {
  let read = false;
  return {
    isFile: false,
    isDirectory: true,
    name,
    createReader: () => ({
      readEntries: (resolve: (entries: FileSystemEntry[]) => void) => {
        resolve(read ? [] : files.map(fakeFileEntry));
        read = true;
      },
    }),
  } as unknown as FileSystemEntry;
};

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
