// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sep } from "@tauri-apps/api/path";
import { open, save } from "@tauri-apps/plugin-dialog";
import { readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";

import { downloadFromBrowser } from "@/runtime/download";
import { ENGINE } from "@/runtime/runtime";

export interface FileFilter {
  name: string;
  extensions: string[];
}

export interface PickedFile {
  name: string;
  read: () => Promise<string>;
}

export interface PickFilesArgs {
  title?: string;
  filters?: FileFilter[];
  multiple?: boolean;
}

const browserAccept = (filters?: FileFilter[]): string | undefined => {
  if (filters == null || filters.length === 0) return undefined;
  return filters
    .flatMap(({ extensions }) => extensions.map((ext) => `.${ext}`))
    .join(",");
};

const pickFilesTauri = async ({
  title,
  filters,
  multiple,
}: PickFilesArgs): Promise<PickedFile[] | null> => {
  const result = await open({
    title,
    filters,
    multiple: multiple ?? false,
    directory: false,
  });
  if (result == null) return null;
  const paths = Array.isArray(result) ? result : [result];
  if (paths.length === 0) return null;
  const separator = sep();
  return paths.map((path) => ({
    name: path.split(separator).pop() ?? path,
    read: () => readTextFile(path),
  }));
};

const pickFilesBrowser = ({
  filters,
  multiple,
}: PickFilesArgs): Promise<PickedFile[] | null> =>
  new Promise((resolve) => {
    const input = document.createElement("input");
    input.type = "file";
    if (multiple) input.multiple = true;
    const accept = browserAccept(filters);
    if (accept != null) input.accept = accept;
    let settled = false;
    const settle = (value: PickedFile[] | null) => {
      if (settled) return;
      settled = true;
      resolve(value);
    };
    input.addEventListener("change", () => {
      const files = input.files;
      if (files == null || files.length === 0) return settle(null);
      settle(
        Array.from(files).map((file) => ({
          name: file.name,
          read: () => file.text(),
        })),
      );
    });
    input.addEventListener("cancel", () => settle(null));
    input.click();
  });

/**
 * Opens a native file picker and returns the selected files as a uniform
 * { name, read } shape. On Tauri this uses the OS dialog and reads via the
 * filesystem plugin; in the browser it uses an <input type="file"> and reads
 * via File.text(). Returns null if the user cancels or selects nothing.
 */
export const pickFiles = (args: PickFilesArgs): Promise<PickedFile[] | null> =>
  ENGINE === "tauri" ? pickFilesTauri(args) : pickFilesBrowser(args);

export interface SaveFileArgs {
  title?: string;
  defaultName: string;
  filters?: FileFilter[];
  contents: string;
}

/**
 * Saves a string to disk via a native save dialog on Tauri, or a download to
 * the browser's downloads folder on the web. Returns a human-readable location
 * for status messages (the chosen path on Tauri, the filename on the browser),
 * or null if the user cancels.
 */
export const saveFile = async ({
  title,
  defaultName,
  filters,
  contents,
}: SaveFileArgs): Promise<string | null> => {
  if (ENGINE === "tauri") {
    const path = await save({ title, defaultPath: defaultName, filters });
    if (path == null) return null;
    await writeTextFile(path, contents);
    return path;
  }
  downloadFromBrowser(new Blob([contents], { type: "application/json" }), defaultName);
  return defaultName;
};
