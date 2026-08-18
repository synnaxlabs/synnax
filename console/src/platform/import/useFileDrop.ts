// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type ontology, type project, type Synnax as Client } from "@synnaxlabs/client";
import { type Mosaic, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import {
  captureEntries,
  isDirectoryEntry,
  isFileEntry,
  readDirectoryFiles,
  readEntryFile,
} from "@/platform/import/entries";
import { type BundleIngester, ingestServer } from "@/platform/import/import";
import { useIngestBatch } from "@/platform/import/useIngestBatch";
import { isZipFile, zipFiles } from "@/platform/import/zip";
import { Session } from "@/session";

const readJSON = async (file: File): Promise<unknown> => JSON.parse(await file.text());

interface IngestContext {
  client: Client | null;
  ingestBundle: BundleIngester;
  projectKey: project.Key;
  store: Store;
}

// Returns the resource the entry created, or nothing for a bundle: a project import
// brings its own panels, so it opens no tab here. A dropped directory is zipped with
// its files' relative paths; the Core owns the bundle format.
const ingestEntry = async (
  entry: FileSystemEntry,
  { client, ingestBundle, projectKey, store }: IngestContext,
): Promise<void | ontology.ID> => {
  if (isDirectoryEntry(entry)) {
    const bundle = await zipFiles(await readDirectoryFiles(entry));
    return await ingestBundle(entry.name, bundle, { client, store });
  }
  if (!isFileEntry(entry)) return;
  const file = await readEntryFile(entry);
  if (isZipFile(file.name))
    return await ingestBundle(file.name, new Uint8Array(await file.arrayBuffer()), {
      client,
      store,
    });
  if (file.type !== "application/json") throw new Error("not a JSON file");
  return await ingestServer(await readJSON(file), {
    client,
    projectKey,
    fileName: file.name,
  });
};

export interface UseFileDropParams {
  /**
   * Ingests a dropped directory or .zip as a bundle. Injected by the composition root.
   */
  ingestBundle: BundleIngester;
}

/**
 * A drop the mosaic reports, or one with no leaf to place into: the Console has no
 * panel open, so the tabs open in a panel created for them.
 */
export interface FileDropProps extends Partial<Mosaic.OnFileDropProps> {
  event: Mosaic.OnFileDropProps["event"];
}

export type FileDrop = (props: FileDropProps) => void;

/**
 * Returns a file drop handler that imports every dropped JSON file and directory
 * concurrently, then opens the resources they created as one batch of tabs in the leaf
 * the drop landed on. A file that fails is reported on its own.
 */
export const useFileDrop = ({ ingestBundle }: UseFileDropParams): FileDrop => {
  const client = Synnax.use();
  const store = Session.useStore();
  const handleError = Status.useErrorHandler();
  const ingestBatch = useIngestBatch();
  return useCallback(
    ({ nodeKey, location, event }: FileDropProps) => {
      const entries = captureEntries(event.dataTransfer);
      // A dropped project selects itself once imported, so every file in the drop takes
      // the project open when it landed instead of whichever one wins the race.
      const projectKey = Session.Project.selectSelected(store.getState());
      const placement =
        nodeKey != null && location != null ? { leaf: nodeKey, location } : undefined;
      handleError(
        async () =>
          await ingestBatch({
            items: entries,
            ingest: async (entry) =>
              await ingestEntry(entry, { client, ingestBundle, projectKey, store }),
            placement,
          }),
      );
    },
    [client, ingestBundle, ingestBatch, store, handleError],
  );
};
