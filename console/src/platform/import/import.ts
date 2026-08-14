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
  type ontology,
  type panel,
  project,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useCallback } from "react";
import { ZodError } from "zod";

import { useFileIngesters } from "@/platform/import/FileIngestersProvider";
import {
  type FileIngester,
  type FileIngesterContext,
  type FileIngesters,
} from "@/platform/import/ingester";
import { trimFileName } from "@/platform/import/trimFileName";
import { Panel } from "@/platform/panel";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

/**
 * Imports data by streaming its bytes to the Core, which owns envelope decoding, type
 * resolution for typeless legacy Console states, legacy-version migration, file-name
 * naming, and project parenting.
 * @throws {DisconnectedError} if no cluster is connected.
 */
export const ingestServer: FileIngester = async (
  data,
  { client, projectKey, fileName },
) => {
  if (client == null) throw new DisconnectedError();
  return await client.imex.import(JSON.stringify(data), {
    encoding: "JSON",
    fileName,
    parent: project.ontologyID(projectKey),
  });
};

/** Maps the resources ingest created to the tabs that open them. */
export const resourceTabs = (ids: (ontology.ID | void)[]): panel.NewTab[] =>
  ids.filter((id) => id != null).map((resource) => ({ variant: "resource", resource }));

export const ingestComponent = async (
  data: unknown,
  fileIngesters: FileIngesters,
  ctx: FileIngesterContext,
): Promise<void | ontology.ID> => {
  let type: string | undefined;
  if (
    typeof data === "object" &&
    data != null &&
    "type" in data &&
    typeof data.type === "string"
  )
    type = data.type;
  if (type != null) {
    const ingest = fileIngesters[type];
    // Types without a client-side ingester are the server's to decode.
    if (ingest != null) return await ingest(data, ctx);
    return await ingestServer(data, ctx);
  }
  // Typeless files are either legacy task configs — client-side ingesters reject
  // foreign payloads with a ZodError — or legacy Console states, which the server
  // recognizes by their frozen markers.
  for (const ingest of Object.values(fileIngesters))
    try {
      return await ingest(data, ctx);
    } catch (e) {
      if (e instanceof ZodError) continue;
      throw errors.fromUnknown(e);
    }
  return await ingestServer(data, ctx);
};

const FILTERS = [{ name: "JSON", extensions: ["json"] }];

interface ImportComponentParams {
  handleError: Status.ErrorHandler;
  client: Client | null;
  openTabs: Panel.OpenTabs;
  store: Store;
  projectKey?: string;
  fileIngesters: FileIngesters;
}

const importComponent = ({
  store,
  client,
  openTabs,
  handleError,
  projectKey,
  fileIngesters,
}: ImportComponentParams): void => {
  handleError(async () => {
    const files = await Runtime.pickFiles({
      title: "Import",
      filters: FILTERS,
      multiple: true,
    });
    if (files == null) return;
    const storeState = store.getState();
    const activeProjectKey = Session.Project.selectSelected(storeState);
    if (projectKey != null && activeProjectKey !== projectKey) {
      if (client == null) throw new DisconnectedError();
      const proj = await client.projects.retrieve(projectKey);
      store.dispatch(Session.Project.select(proj.key));
    }
    const activeProjectKeyAfter = Session.Project.selectSelected(store.getState());
    const ids = await Promise.all(
      files.map(async (file) => {
        try {
          const data = await file.read();
          return await ingestComponent(JSON.parse(data), fileIngesters, {
            name: trimFileName(file.name),
            client,
            projectKey: activeProjectKeyAfter,
            fileName: file.name,
          });
        } catch (e) {
          handleError(e, `Failed to import ${file.name}`);
        }
      }),
    );
    openTabs(resourceTabs(ids));
  });
};

export const useImport = (): ((projectKey?: string) => void) => {
  const openTabs = Panel.useOpenTabs();
  const store = Session.useStore();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const fileIngesters = useFileIngesters();
  return useCallback(
    (projectKey?: string) =>
      importComponent({
        store,
        openTabs,
        client,
        handleError,
        projectKey,
        fileIngesters,
      }),
    [store, openTabs, client, handleError, fileIngesters],
  );
};
