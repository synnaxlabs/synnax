// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useCallback } from "react";
import { ZodError } from "zod";

import { useFileIngesters } from "@/platform/import/FileIngestersProvider";
import {
  type FileIngesterContext,
  type FileIngesters,
} from "@/platform/import/ingester";
import { trimFileName } from "@/platform/import/trimFileName";
import { Panel } from "@/platform/panel";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

export const ingestComponent = async (
  data: unknown,
  fileIngesters: FileIngesters,
  ctx: FileIngesterContext,
): Promise<void> => {
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
    await ingest(data, ctx);
    return;
  }
  for (const ingest of Object.values(fileIngesters))
    try {
      await ingest(data, ctx);
      return;
    } catch (e) {
      if (e instanceof ZodError) continue;
      else throw errors.fromUnknown(e);
    }
  throw new Error(`${ctx.fileName} cannot be imported.`);
};

const FILTERS = [{ name: "JSON", extensions: ["json"] }];

interface ImportComponentParams {
  handleError: Status.ErrorHandler;
  client: Client | null;
  openTab: Panel.OpenTab;
  store: Store;
  projectKey?: string;
  fileIngesters: FileIngesters;
}

const importComponent = ({
  store,
  client,
  openTab,
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
    files.forEach((file) =>
      handleError(async () => {
        const data = await file.read();
        const name = trimFileName(file.name);
        await ingestComponent(JSON.parse(data), fileIngesters, {
          name,
          openTab,
          client,
          projectKey: activeProjectKeyAfter,
          fileName: file.name,
        });
      }, `Failed to import ${file.name}`),
    );
  });
};

export const useImport = (): ((projectKey?: string) => void) => {
  const openTab = Panel.useOpenTab();
  const store = Session.useStore();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const fileIngesters = useFileIngesters();
  return useCallback(
    (projectKey?: string) =>
      importComponent({
        store,
        openTab,
        client,
        handleError,
        projectKey,
        fileIngesters,
      }),
    [store, openTab, client, handleError, fileIngesters],
  );
};
