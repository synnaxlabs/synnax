// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import {
  DisconnectedError,
  imex,
  type ontology,
  project,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useImportBatch } from "@/platform/import/useImportBatch";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

export interface ImportServerContext {
  client: Client | null;
  /** The ontology resource the created resource is parented to. */
  parent: ontology.ID;
  /**
   * The name of the file the data was read from, extension included. The Core names
   * the resource after the file when the file's contents carry no name.
   */
  fileName: string;
}

interface BundleImporterContext {
  client: Client | null;
  store: Store;
}

/**
 * Imports a zipped bundle read from a picked or dropped source. name is the source
 * archive or folder's name, the Core's fallback for naming the imported resource.
 */
export interface BundleImporter {
  (
    name: string,
    bundle: Uint8Array<ArrayBuffer>,
    ctx: BundleImporterContext,
  ): Promise<void>;
}

/**
 * Imports a resource by streaming its serialized bytes to the Core, which owns envelope
 * decoding, type resolution for typeless legacy Console states, legacy-version
 * migration, file-name naming, and parenting. Returns the created resource's ID.
 * @throws {DisconnectedError} if no Core is connected.
 */
export const importServer = async (
  data: Uint8Array<ArrayBuffer>,
  { client, parent, fileName }: ImportServerContext,
): Promise<ontology.ID> => {
  if (client == null) throw new DisconnectedError();
  return await client.imex.import(data, { ...imex.JSON_OPTIONS, fileName, parent });
};

export const useImport = (): ((projectKey?: string) => void) => {
  const store = Session.useStore();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const importBatch = useImportBatch();
  return useCallback(
    (projectKey?: string) =>
      handleError(async () => {
        const files = await Runtime.pickFiles({
          title: "Import",
          extension: "json",
          multiple: true,
        });
        if (files == null) return;
        const activeProjectKey = Session.Project.selectSelected(store.getState());
        if (projectKey != null && activeProjectKey !== projectKey) {
          if (client == null) throw new DisconnectedError();
          const proj = await client.projects.retrieve(projectKey);
          store.dispatch(Session.Project.select(proj.key));
        }
        const activeProjectKeyAfter = Session.Project.selectSelected(store.getState());
        await importBatch({
          // importBatch names failures after the item, so the path doubles as the name.
          items: files.map((file) => ({ name: file.path, readBytes: file.readBytes })),
          importItem: async (file) =>
            await importServer(await file.readBytes(), {
              client,
              parent: project.ontologyID(activeProjectKeyAfter),
              fileName: file.name,
            }),
        });
      }),
    [store, client, handleError, importBatch],
  );
};
