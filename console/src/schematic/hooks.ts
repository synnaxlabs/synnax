// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { type DialogFilter, open, save } from "@tauri-apps/plugin-dialog";
import { readFile, writeFile } from "@tauri-apps/plugin-fs";
import { useDispatch, useStore } from "react-redux";

import { Confirm } from "@/confirm";
import { Layout } from "@/layout";
import { fileHandler } from "@/schematic/file";
import { select } from "@/schematic/selectors";
import { type State, type StateWithName } from "@/schematic/slice";
import { RootState } from "@/store";
import { Workspace } from "@/workspace";

const filters: DialogFilter[] = [
  {
    name: "JSON",
    extensions: ["json"],
  },
];

export const useExport = (name: string = "schematic"): ((key: string) => void) => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();

  return useMutation<void, Error, string>({
    mutationFn: async (key) => {
      const storeState = store.getState();
      let state = select(storeState, key);
      let name = Layout.select(storeState, key)?.name;
      if (state == null) {
        if (client == null) throw new UnexpectedError("Client is unavailable");
        const schematic = await client.workspaces.schematic.retrieve(key);
        state = {
          ...(schematic.data as unknown as State),
          snapshot: schematic.snapshot,
          key: schematic.key,
        };
        name = schematic.name;
      }
      if (name == null)
        throw new UnexpectedError("Schematic with key is missing in store state");
      const savePath = await save({
        title: `Export ${name}`,
        defaultPath: `${name}.json`,
        filters,
      });
      if (savePath == null) return;
      const schematicData: StateWithName = { ...state, name };
      await writeFile(
        savePath,
        new TextEncoder().encode(JSON.stringify(schematicData)),
      );
    },
    onError: (err) =>
      addStatus({
        variant: "error",
        message: `Failed to export ${name}`,
        description: err.message,
      }),
  }).mutate;
};

export const useImport = (workspaceKey?: string): (() => void) => {
  const placer = Layout.usePlacer();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();
  const confirm = Confirm.useModal();
  const client = Synnax.use();
  const dispatch = useDispatch();
  const activeKey = Workspace.useSelectActiveKey();
  if (workspaceKey != null && activeKey !== workspaceKey)
    dispatch(Workspace.setActive(workspaceKey));

  let name = "schematic";

  return useMutation<void, Error>({
    mutationFn: async () => {
      const fileResponses = await open({
        title: "Import schematic",
        filters,
        multiple: true,
        directory: false,
      });
      if (fileResponses == null) return;
      for (const fileResponse of fileResponses) {
        if (fileResponse == null) return;
        const rawData = await readFile(fileResponse.path);
        const fileName = fileResponse.path.split("/").pop();
        if (fileName == null) throw new UnexpectedError("File name is null");
        name = fileName;
        const file = JSON.parse(new TextDecoder().decode(rawData));
        if (
          !(await fileHandler({
            file,
            placer,
            name: fileName,
            store,
            confirm,
            client,
            workspaceKey,
            dispatch,
          }))
        )
          addStatus({
            variant: "error",
            message: `Failed to import ${fileName}`,
            description: `${fileName} is an invalid schematic`,
          });
      }
    },
    onError: (err) =>
      addStatus({
        variant: "error",
        message: `Failed to import ${name}`,
        description: err.message,
      }),
  }).mutate;
};
