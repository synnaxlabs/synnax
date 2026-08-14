// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type ontology, type Synnax as Client } from "@synnaxlabs/client";
import { type Mosaic, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useFileIngesters } from "@/platform/import/FileIngestersProvider";
import { ingestComponent, resourceTabs } from "@/platform/import/import";
import { type DirectoryIngester, type FileIngesters } from "@/platform/import/ingester";
import { trimFileName } from "@/platform/import/trimFileName";
import { Panel } from "@/platform/panel";
import { Session } from "@/session";

// DataTransferItem handles stop resolving once the drop handler returns, so the entries
// are taken before anything is awaited.
const captureEntries = (dataTransfer: DataTransfer): FileSystemEntry[] =>
  Array.from(dataTransfer.items)
    .filter((item) => item.kind === "file")
    .map((item) => item.webkitGetAsEntry())
    .filter((entry) => entry != null);

// The DOM entry types carry isFile and isDirectory as plain booleans, so narrowing to
// the reading APIs takes a guard.
const isFile = (entry: FileSystemEntry): entry is FileSystemFileEntry => entry.isFile;

const isDirectory = (entry: FileSystemEntry): entry is FileSystemDirectoryEntry =>
  entry.isDirectory;

const readFile = async (entry: FileSystemFileEntry): Promise<File> =>
  await new Promise((resolve, reject) => entry.file(resolve, reject));

const readDirectory = async (entry: FileSystemDirectoryEntry): Promise<File[]> => {
  const reader = entry.createReader();
  const files: File[] = [];
  while (true) {
    const entries = await new Promise<FileSystemEntry[]>((resolve, reject) => {
      reader.readEntries(resolve, reject);
    });
    if (entries.length === 0) break;
    for (const child of entries) if (isFile(child)) files.push(await readFile(child));
  }
  return files;
};

const readJSON = async (file: File): Promise<unknown> => JSON.parse(await file.text());

interface IngestContext {
  client: Client | null;
  fileIngesters: FileIngesters;
  ingestDirectory: DirectoryIngester;
  store: Store;
}

// Returns the resource the entry created, or nothing for a directory: a project import
// brings its own panels, so it opens no tab here.
const ingestEntry = async (
  entry: FileSystemEntry,
  { client, fileIngesters, ingestDirectory, store }: IngestContext,
): Promise<void | ontology.ID> => {
  if (isDirectory(entry)) {
    const files = await readDirectory(entry);
    const parsed = await Promise.all(
      files.map(async (file) => ({ name: file.name, data: await readJSON(file) })),
    );
    return await ingestDirectory(entry.name, parsed, {
      client,
      fileIngesters,
      store,
    });
  }
  if (!isFile(entry)) return;
  const file = await readFile(entry);
  if (file.type !== "application/json") throw new Error("not a JSON file");
  return await ingestComponent(await readJSON(file), fileIngesters, {
    name: trimFileName(file.name),
    client,
    projectKey: Session.Project.selectSelected(store.getState()),
    fileName: file.name,
  });
};

export interface UseFileDropParams {
  /** Ingests a dropped directory. Injected by the composition root. */
  ingestDirectory: DirectoryIngester;
}

/**
 * Returns a mosaic file drop handler that imports every dropped JSON file and directory
 * concurrently, then opens the resources they created as one batch of tabs in the leaf
 * the drop landed on. A file that fails is reported on its own.
 */
export const useFileDrop = ({
  ingestDirectory,
}: UseFileDropParams): ((props: Mosaic.OnFileDropProps) => void) => {
  const client = Synnax.use();
  const store = Session.useStore();
  const openTabs = Panel.useOpenTabs();
  const handleError = Status.useErrorHandler();
  const fileIngesters = useFileIngesters();
  return useCallback(
    ({ nodeKey, location, event }: Mosaic.OnFileDropProps) => {
      const entries = captureEntries(event.dataTransfer);
      handleError(async () => {
        const ids = await Promise.all(
          entries.map(async (entry) => {
            try {
              return await ingestEntry(entry, {
                client,
                fileIngesters,
                ingestDirectory,
                store,
              });
            } catch (e) {
              handleError(e, `Failed to import ${entry.name}`);
            }
          }),
        );
        openTabs(resourceTabs(ids), { placement: { leaf: nodeKey, location } });
      });
    },
    [client, fileIngesters, ingestDirectory, openTabs, store, handleError],
  );
};
