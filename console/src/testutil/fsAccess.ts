// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// jsdom does not implement the File System Access API, so window carries neither
// picker. Declare them as optional members (matching the production declaration for
// showDirectoryPicker in @/platform/runtime/files) so specs can install and remove
// fakes without re-casting window.
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

/** installSaveFilePicker installs picker as window.showSaveFilePicker. */
export const installSaveFilePicker = (
  picker: NonNullable<Window["showSaveFilePicker"]>,
): void => {
  window.showSaveFilePicker = picker;
};

/** installDirectoryPicker installs picker as window.showDirectoryPicker. */
export const installDirectoryPicker = (
  picker: NonNullable<Window["showDirectoryPicker"]>,
): void => {
  window.showDirectoryPicker = picker;
};

/** removeFilePickers removes both File System Access pickers from window. */
export const removeFilePickers = (): void => {
  delete window.showSaveFilePicker;
  delete window.showDirectoryPicker;
};

export interface InstallFakeDirectoryPickerArgs {
  name?: string;
  preExisted?: boolean;
}

export interface FakeDirectoryPicker {
  /** Files written through the picker, keyed by filename. */
  files: Map<string, string>;
}

/**
 * Installs a window.showDirectoryPicker returning a minimal writable directory
 * handle that records written files. jsdom cannot construct real
 * FileSystemDirectoryHandle instances, so the sanctioned casts to the DOM types
 * live here.
 */
export const installFakeDirectoryPicker = ({
  name = "exports",
  preExisted = false,
}: InstallFakeDirectoryPickerArgs = {}): FakeDirectoryPicker => {
  const files = new Map<string, string>();
  const createSubdir = (subName: string): FileSystemDirectoryHandle =>
    ({
      name: subName,
      getFileHandle: async (fileName: string) => ({
        createWritable: async () => {
          let text = "";
          return {
            write: async (chunk: string) => {
              text += chunk;
            },
            close: async () => {
              files.set(fileName, text);
            },
          };
        },
      }),
    }) as unknown as FileSystemDirectoryHandle;
  const root = {
    name,
    getDirectoryHandle: async (subName: string, opts?: { create?: boolean }) => {
      if (!preExisted && opts?.create !== true)
        throw new DOMException("directory not found", "NotFoundError");
      return createSubdir(subName);
    },
  } as unknown as FileSystemDirectoryHandle;
  installDirectoryPicker(async () => root);
  return { files };
};

export interface InstallPickedDirectoryOptions {
  /**
   * Whether the export subdirectory already exists in the picked directory. When
   * false, the first lookup rejects with NotFoundError so the export proceeds
   * without a replace confirmation.
   */
  exists?: boolean;
}

/**
 * Installs a fake FS Access directory picker whose picked directory records every
 * file write into the returned map, keyed by file name. The fake handles cover the
 * minimal surface the project export path touches; jsdom cannot construct real
 * FileSystemDirectoryHandles, so the sanctioned cast lives here.
 */
export const installPickedDirectory = ({
  exists = true,
}: InstallPickedDirectoryOptions = {}): Map<string, string> => {
  const writes = new Map<string, string>();
  const subHandle = {
    getFileHandle: async (name: string) => ({
      createWritable: async () => ({
        write: async (data: string) => void writes.set(name, data),
        close: async () => {},
      }),
    }),
  };
  let firstLookup = true;
  const root = {
    name: "Downloads",
    getDirectoryHandle: async () => {
      if (!exists && firstLookup) {
        firstLookup = false;
        throw new DOMException("missing", "NotFoundError");
      }
      return subHandle;
    },
  };
  installDirectoryPicker(async () => root as unknown as FileSystemDirectoryHandle);
  return writes;
};
