// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";
import { ZodError } from "zod";

import { type DirectoryIngestor, type FileIngestor } from "@/import/ingestor";
import { trimFileName } from "@/import/trimFileName";
import { type Layout } from "@/layout";

interface DirectoryContent {
  name: string;
  files: File[];
}

const parseDataTransferItem = async (
  item: DataTransferItem,
): Promise<File | DirectoryContent | null> => {
  if (item.kind !== "file") return null;

  const entry = item.webkitGetAsEntry();
  if (!entry) return null;

  if (entry.isFile) return item.getAsFile();
  if (!entry.isDirectory) return null;

  const directoryReader = (entry as FileSystemDirectoryEntry).createReader();
  const files: File[] = [];

  const processEntries = async (entries: FileSystemEntry[]): Promise<void> => {
    await Promise.all(
      entries.map(async (entry) => {
        if (entry.isFile) {
          const file = await new Promise<File | null>((resolve) => {
            (entry as FileSystemFileEntry).file(resolve, () => resolve(null));
          });
          if (file) files.push(file);
        }
      }),
    );
  };

  const readAllEntries = async (): Promise<void> => {
    while (true) {
      const entries = await new Promise<FileSystemEntry[]>((resolve, reject) => {
        directoryReader.readEntries(resolve, reject);
      });
      if (entries.length === 0) break;
      await processEntries(entries);
    }
  };

  await readAllEntries();
  return { name: entry.name, files };
};

interface DataTransferItemContext {
  client: Synnax | null;
  fileIngestors: Record<string, FileIngestor>;
  ingestDirectory: DirectoryIngestor;
  layout: Partial<Layout.State>;
  placeLayout: Layout.Placer;
  store: Store;
}

export const dataTransferItem = async (
  item: DataTransferItem,
  {
    client,
    fileIngestors,
    ingestDirectory,
    layout,
    placeLayout,
    store,
  }: DataTransferItemContext,
) => {
  const entry = await parseDataTransferItem(item);
  if (entry == null) throw new Error("path is null");

  // Handling a file transfer, importing a single JSON file
  if (entry instanceof File) {
    const name = trimFileName(entry.name);
    if (entry.type !== "application/json") throw new Error("not a JSON file");
    const buffer = await entry.arrayBuffer();
    const fileData = new TextDecoder().decode(buffer);
    let hasBeenIngested = false;
    for (const ingest of Object.values(fileIngestors))
      try {
        ingest(fileData, {
          layout: { ...layout, name },
          placeLayout,
          store,
        });
        hasBeenIngested = true;
        break;
      } catch (e) {
        if (e instanceof ZodError) continue;
        else throw e;
      }
    if (!hasBeenIngested) throw new Error(`${entry.name} is not a valid layout file`);
    return;
  }

  // Handling a directory transfer, importing a directory containing multiple files
  const parsedFiles = await Promise.all(
    entry.files.map(async (file) => {
      const buffer = await file.arrayBuffer();
      const data = new TextDecoder().decode(buffer);
      return { name: file.name, data };
    }),
  );
  await ingestDirectory(entry.name, parsedFiles, {
    client,
    ingestors: fileIngestors,
    placeLayout,
    store,
  });
};
