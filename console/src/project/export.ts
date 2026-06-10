// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { useStore } from "react-redux";

import { Export } from "@/export";
import { useExtractors } from "@/export/ExtractorsProvider";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { purgeExcludedLayouts } from "@/project/purgeExcludedLayouts";
import { selectActive } from "@/project/selectors";
import { Runtime } from "@/runtime";
import { type RootAction, type RootState, type RootStore } from "@/store";

export interface ExportContext {
  client: Client | null;
  store: RootStore;
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
    const active = selectActive(storeState);
    let toExport: Layout.SliceState;
    if (active?.key === key || key == null) {
      const file = Layout.selectSliceState(storeState);
      toExport = purgeExcludedLayouts(file);
      if (active?.key != null) name = active.name;
    } else {
      if (client == null) throw new DisconnectedError();
      const ws = await client.projects.retrieve(key);
      toExport = ws.layout as Layout.SliceState;
      name = ws.name;
    }
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
    const namesSet = new Set<string>();
    Object.values(toExport.layouts).forEach((layout) => {
      const deduplicatedName = strings.deduplicateFileName(layout.name, namesSet);
      layout.name = Export.sanitizeFileName(deduplicatedName);
      namesSet.add(layout.name);
    });
    await directory.writeText(LAYOUT_FILE_NAME, JSON.stringify(toExport));
    const fileInfos: Export.File[] = [];
    await Promise.all(
      Object.values(toExport.layouts).map(async ({ type, key }) => {
        const extractor = extractors[type];
        if (extractor == null) return;
        const { data } = await extractor(key, { store, client });
        fileInfos.push({ data, name: `${toExport.layouts[key].name}.json` });
      }),
    );
    await Promise.all(
      fileInfos.map(({ data, name }) => directory.writeText(name, data)),
    );
    addStatus({
      variant: "success",
      message: `Exported ${name} to ${directory.displayPath}`,
    });
  }, `Failed to export ${name}`);
};

export const LAYOUT_FILE_NAME = "LAYOUT.json";

export const useExport = (): ((key: string | null) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const store = useStore<RootState, RootAction>();
  const confirm = Modals.useConfirm();
  const extractors = useExtractors();
  return (key: string | null) =>
    export_(key, { client, store, confirm, handleError, extractors, addStatus });
};
