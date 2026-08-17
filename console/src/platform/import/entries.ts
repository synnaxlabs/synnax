// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Takes the file-system entries out of a drop's data transfer. Call it before anything
 * awaits: DataTransferItem handles stop resolving once the drop handler returns.
 */
export const captureEntries = (dataTransfer: DataTransfer): FileSystemEntry[] =>
  Array.from(dataTransfer.items)
    .filter((item) => item.kind === "file")
    .map((item) => item.webkitGetAsEntry())
    .filter((entry) => entry != null);

// The DOM entry types carry isFile and isDirectory as plain booleans, so narrowing to
// the reading APIs takes a guard.
export const isFileEntry = (entry: FileSystemEntry): entry is FileSystemFileEntry =>
  entry.isFile;

export const isDirectoryEntry = (
  entry: FileSystemEntry,
): entry is FileSystemDirectoryEntry => entry.isDirectory;

/** Reads a dropped file entry into the File it points at. */
export const readEntryFile = async (entry: FileSystemFileEntry): Promise<File> =>
  await new Promise((resolve, reject) => entry.file(resolve, reject));

/** Reads a dropped directory entry's top-level files, skipping subdirectories. */
export const readDirectoryFiles = async (
  entry: FileSystemDirectoryEntry,
): Promise<File[]> => {
  const reader = entry.createReader();
  const files: File[] = [];
  while (true) {
    const entries = await new Promise<FileSystemEntry[]>((resolve, reject) => {
      reader.readEntries(resolve, reject);
    });
    if (entries.length === 0) break;
    for (const child of entries)
      if (isFileEntry(child)) files.push(await readEntryFile(child));
  }
  return files;
};
