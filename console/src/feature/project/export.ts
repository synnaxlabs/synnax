// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  arc,
  DisconnectedError,
  lineplot,
  log,
  type ontology,
  type panel,
  project,
  schematic,
  type Synnax as Client,
  table,
  task,
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
  const children = await client.ontology.retrieveChildren(
    project.ontologyID(projectKey),
  );
  const keys = children.filter(({ id }) => id.type === "panel").map(({ id }) => id.key);
  if (keys.length === 0) return [];
  return await client.panels.retrieve(keys);
};

// HACK: a client-side mirror of the resource types the Core registers exporters for. A
// panel tab can reference types the Core cannot export (e.g. ranges), and the Core has
// no endpoint to ask which types are exportable, so we hardcode the list here. Keep in
// sync with the RegisterExporter calls in core/pkg/service until such an endpoint
// exists.
const EXPORTABLE_TYPES = new Set<string>([
  arc.TYPE_ONTOLOGY_ID.type,
  lineplot.TYPE_ONTOLOGY_ID.type,
  log.TYPE_ONTOLOGY_ID.type,
  schematic.TYPE_ONTOLOGY_ID.type,
  table.TYPE_ONTOLOGY_ID.type,
  task.TYPE_ONTOLOGY_ID.type,
]);

export interface ExportContext {
  client: Client | null;
  store: Store;
  confirm: Modals.PromptConfirm;
  handleError: Status.ErrorHandler;
  addStatus: Status.Adder;
}

export const export_ = (
  key: string | null,
  { client, store, confirm, handleError, addStatus }: ExportContext,
): void => {
  let name: string = "project"; // default name for error message
  handleError(async () => {
    const storeState = store.getState();
    const targetKey = key ?? Session.Project.selectSelected(storeState);
    if (client == null) throw new DisconnectedError();
    const proj = await client.projects.retrieve(targetKey);
    name = proj.name;
    const panels = await retrievePanels(client, targetKey);
    const directory = await Runtime.pickWritableDirectory({
      title: `Select a location to export ${name}`,
      subdirectory: strings.sanitizeFileName(name),
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
      Array.from(resources.values())
        .filter(({ type }) => EXPORTABLE_TYPES.has(type))
        .map(async (id) => {
          const file = await Export.fetchFile(client, id);
          const fileName = strings.sanitizeFileName(
            strings.deduplicateFileName(file.name, namesSet),
          );
          namesSet.add(fileName);
          fileInfos.push({ data: file.data, name: fileName });
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
  return (key: string | null) =>
    export_(key, { client, store, confirm, handleError, addStatus });
};
