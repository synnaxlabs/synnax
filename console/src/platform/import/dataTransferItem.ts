// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";

import { ingestComponent } from "@/platform/import/import";
import { type DirectoryIngester, type FileIngesters } from "@/platform/import/ingester";
import { trimFileName } from "@/platform/import/trimFileName";
import { type Panel } from "@/platform/panel";
import { Session } from "@/session";

interface DirectoryFile {
  file: File;
  /** The file's path relative to the dropped directory, forward-slash form. */
  path: string;
}

interface DirectoryContent {
  name: string;
  files: DirectoryFile[];
}

const parseDataTransferItem = async (
  item: DataTransferItem,
): Promise<File | DirectoryContent | null> => {
  if (item.kind !== "file") return null;

  const entry = item.webkitGetAsEntry();
  if (!entry) return null;

  if (entry.isFile) return item.getAsFile();
  if (!entry.isDirectory) return null;

  const files: DirectoryFile[] = [];

  const readDirectory = async (
    dir: FileSystemDirectoryEntry,
    prefix: string,
  ): Promise<void> => {
    const reader = dir.createReader();
    while (true) {
      const entries = await new Promise<FileSystemEntry[]>((resolve, reject) => {
        reader.readEntries(resolve, reject);
      });
      if (entries.length === 0) break;
      await Promise.all(
        entries.map(async (child) => {
          const path = prefix === "" ? child.name : `${prefix}/${child.name}`;
          if (child.isDirectory)
            return await readDirectory(child as FileSystemDirectoryEntry, path);
          if (!child.isFile) return;
          const file = await new Promise<File | null>((resolve) => {
            (child as FileSystemFileEntry).file(resolve, () => resolve(null));
          });
          if (file) files.push({ file, path });
        }),
      );
    }
  };

  await readDirectory(entry as FileSystemDirectoryEntry, "");
  return { name: entry.name, files };
};

interface DataTransferItemContext {
  client: Synnax | null;
  fileIngesters: FileIngesters;
  ingestDirectory: DirectoryIngester;
  openTab: Panel.OpenTab;
  store: Store;
}

export const dataTransferItem = async (
  item: DataTransferItem,
  { client, fileIngesters, ingestDirectory, openTab, store }: DataTransferItemContext,
) => {
  const entry = await parseDataTransferItem(item);
  if (entry == null) throw new Error("path is null");

  // Handling a file transfer, importing a single JSON file
  if (entry instanceof File) {
    const name = trimFileName(entry.name);
    if (entry.type !== "application/json") throw new Error("not a JSON file");
    const buffer = await entry.arrayBuffer();
    const fileData = new TextDecoder().decode(buffer);
    const parsedData = JSON.parse(fileData);
    const projectKey = Session.Project.selectSelected(store.getState());
    await ingestComponent(parsedData, fileIngesters, {
      name,
      openTab,
      client,
      projectKey,
      fileName: entry.name,
    });
    return;
  }

  // Handling a directory transfer, importing a directory containing multiple files
  const parsedFiles = await Promise.all(
    entry.files.map(async ({ file, path }) => {
      const buffer = await file.arrayBuffer();
      const data = new TextDecoder().decode(buffer);
      return { name: file.name, path, data: JSON.parse(data) };
    }),
  );
  await ingestDirectory(entry.name, parsedFiles, {
    client,
    fileIngesters,
    openTab,
    store,
  });
};
