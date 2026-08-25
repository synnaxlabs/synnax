// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// jsdom does not implement the File System Access API, so window carries no save
// picker. Declare it as an optional member so specs can install and remove fakes
// without re-casting window.
declare global {
  interface Window {
    showSaveFilePicker?: (options: {
      suggestedName?: string;
    }) => Promise<FileSystemFileHandle>;
  }
}

/**
 * fakeSaveFileHandle builds the minimal FileSystemFileHandle surface the download path
 * touches (name + createWritable). jsdom cannot construct a real FileSystemFileHandle,
 * so the single sanctioned cast to the DOM type lives here.
 */
export const fakeSaveFileHandle = (
  name: string,
  writable: WritableStream<Uint8Array> = new WritableStream({ write() {} }),
): FileSystemFileHandle =>
  ({ name, createWritable: async () => writable }) as unknown as FileSystemFileHandle;

export const installSaveFilePicker = (
  picker: NonNullable<Window["showSaveFilePicker"]>,
): void => {
  window.showSaveFilePicker = picker;
};

/** removeSaveFilePicker removes the File System Access save picker from window. */
export const removeSaveFilePicker = (): void => {
  delete window.showSaveFilePicker;
};
