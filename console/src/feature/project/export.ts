// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  type ontology,
  type panel,
  project,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";

import { Export } from "@/platform/export";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";
import { type Store } from "@/session/store";

/** The file inside an exported project directory holding its panel documents. */
export const PANELS_FILE_NAME = "PANELS.json";

const collectResources = (node: panel.Node, out: ontology.ID[]): void => {
  if (node.variant === "leaf") {
    node.tabs.forEach((tab) => {
      if (tab.variant === "resource") out.push(tab.resource);
    });
    return;
  }
  collectResources(node.first, out);
  collectResources(node.last, out);
};

const retrievePanels = async (
  client: Client,
  projectKey: project.Key,
): Promise<panel.Panel[]> => {
  const children = await client.ontology.children.retrieve({
    ids: project.ontologyID(projectKey),
  });
  const keys = children.filter(({ id }) => id.type === "panel").map(({ id }) => id.key);
  if (keys.length === 0) return [];
  return await client.panels.retrieve({ keys });
};

export interface ExportContext {
  client: Client | null;
  store: Store;
  confirm: Modals.PromptConfirm;
  handleError: Status.ErrorHandler;
  extractors: Export.Extractors;
  addStatus: Status.Adder;
}

export const export_ = (
  key: string | null,
  { client, store, confirm, handleError, extractors, addStatus }: ExportContext,
): void => {
  let name: string = "project"; // default name for error message
  handleError(async () => {
    const storeState = store.getState();
    const targetKey = key ?? Session.Project.selectSelected(storeState);
    if (client == null) throw new DisconnectedError();
    const proj = await client.projects.retrieve({ key: targetKey });
    name = proj.name;
    const panels = await retrievePanels(client, targetKey);
    const directory = await Runtime.pickWritableDirectory({
      title: `Select a location to export ${name}`,
      subdirectory: Export.sanitizeFileName(name),
    });
    if (directory == null) return;
    if (
      directory.preExisted &&
      !(await confirm({
        message: `A file or directory already exists at ${directory.displayPath}`,
        description: "Replacing will cause the old data to be deleted.",
        cancel: { label: "Cancel" },
        confirm: { label: "Replace", variant: "error" },
      }))
    )
      return;
    const resources = new Map<string, ontology.ID>();
    panels.forEach(({ root }) => {
      const ids: ontology.ID[] = [];
      collectResources(root, ids);
      ids.forEach((id) => resources.set(`${id.type}:${id.key}`, id));
    });
    const namesSet = new Set<string>();
    const fileInfos: Export.File[] = [];
    await Promise.all(
      Array.from(resources.values()).map(async ({ type, key }) => {
        const extractor = extractors[type];
        if (extractor == null) return;
        const { data, name } = await extractor(key, { store, client });
        const fileName = Export.sanitizeFileName(
          strings.deduplicateFileName(name, namesSet),
        );
        namesSet.add(fileName);
        fileInfos.push({ data, name: fileName });
      }),
    );
    await directory.writeText(PANELS_FILE_NAME, JSON.stringify(panels));
    await Promise.all(
      fileInfos.map(({ data, name }) => directory.writeText(`${name}.json`, data)),
    );
    addStatus({
      variant: "success",
      message: `Exported ${name} to ${directory.displayPath}`,
    });
  }, `Failed to export ${name}`);
};

export const useExport = (): ((key: string | null) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const store = Session.useStore();
  const confirm = Modals.useConfirm();
  const extractors = Export.useExtractors();
  return (key: string | null) =>
    export_(key, { client, store, confirm, handleError, extractors, addStatus });
};
