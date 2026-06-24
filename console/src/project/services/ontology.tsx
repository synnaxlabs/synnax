// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  lineplot,
  log,
  type ontology,
  project,
  schematic,
  table,
} from "@synnaxlabs/client";
import {
  Access,
  Icon,
  LinePlot as PLinePlot,
  Log as PLog,
  Menu,
  Project as Base,
  Schematic as PSchematic,
  Table as PTable,
} from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Import } from "@/import";
import { LinePlot } from "@/layered/service/lineplot";
import { Schematic } from "@/layered/service/schematic";
import { Table } from "@/layered/service/table";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Log } from "@/log";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";
import { useExport } from "@/project/export";
import { selectOptionalActiveKey } from "@/project/selectors";
import { maybeRename, setActive } from "@/project/slice";

const useDelete = createUseDelete({
  type: "Project",
  query: Base.useDelete,
  convertKey: String,
  afterSuccess: ({ data, store }) => {
    const s = store.getState();
    const activeKey = selectOptionalActiveKey(s);
    const active = array.toArray(data).find((k) => k === activeKey);
    if (active == null) return;
    store.dispatch(setActive(null));
    store.dispatch(Layout.clearProject());
  },
});

const useRename = createUseRename({
  query: Base.useRename,
  ontologyID: project.ontologyID,
  convertKey: String,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    store.dispatch(maybeRename({ key, name }));
    rollbacks.push(() => store.dispatch(maybeRename({ key, name: oldName })));
    return { ...data, name };
  },
});

const TreeContextMenu: Ontology.TreeContextMenu = (props): ReactElement => {
  const {
    selection,
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete(props);
  const group = Group.useCreateFromSelection();
  const projectKey = ids[0].key;
  const createPlot = LinePlot.useCreate({ project: projectKey });
  const createLog = Log.useCreate({ project: projectKey });
  const createTable = Table.useCreate({ project: projectKey });
  const firstID = selection.ids[0];
  const createSchematic = Schematic.useCreate({ project: projectKey });
  const importComponent = Import.useImport();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = useExport();
  const handleRename = useRename(props);
  const resources = getResource(ids);
  const first = resources[0];
  const singleResource = resources.length === 1;
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasLinePlotCreatePermission = Access.useCreateGranted(
    lineplot.TYPE_ONTOLOGY_ID,
  );
  const hasLogCreatePermission = Access.useCreateGranted(log.TYPE_ONTOLOGY_ID);
  const hasTableCreatePermission = Access.useCreateGranted(table.TYPE_ONTOLOGY_ID);
  const hasSchematicCreatePermission = Access.useCreateGranted(
    schematic.TYPE_ONTOLOGY_ID,
  );
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && singleResource && (
        <>
          <ContextMenu.RenameItem onClick={handleRename} />
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          shape={shape}
          rootID={rootID}
          onClick={() => group(props)}
        />
      )}
      {hasUpdatePermission || (hasDeletePermission && <Menu.Divider />)}
      {singleResource && (
        <>
          {hasLinePlotCreatePermission && (
            <Menu.Item itemKey="createPlot" onClick={() => createPlot()}>
              <PLinePlot.CreateIcon />
              Create line plot
            </Menu.Item>
          )}
          {hasLogCreatePermission && (
            <Menu.Item itemKey="createLog" onClick={() => createLog()}>
              <PLog.CreateIcon />
              Create log
            </Menu.Item>
          )}
          {hasTableCreatePermission && (
            <Menu.Item itemKey="createTable" onClick={() => createTable()}>
              <PTable.CreateIcon />
              Create table
            </Menu.Item>
          )}
          {hasSchematicCreatePermission && (
            <Menu.Item itemKey="createSchematic" onClick={() => createSchematic()}>
              <PSchematic.CreateIcon />
              Create schematic
            </Menu.Item>
          )}
          <Menu.Divider />
          {hasUpdatePermission && (
            <Menu.Item itemKey="import" onClick={() => importComponent(firstID.key)}>
              <Icon.Import />
              Import component(s)
            </Menu.Item>
          )}
          <Menu.Divider />
          <Export.ContextMenuItem onClick={() => handleExport(first.id.key)} />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, ontologyID: first.id })}
          />
          <Ontology.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const handleSelect: Ontology.HandleSelect = ({
  selection,
  client,
  store,
  handleError,
}) => {
  const names = strings.naturalLanguageJoin(
    selection.map(({ name }) => name),
    "project",
  );
  handleError(async () => {
    const proj = await client.projects.retrieve(selection[0].id.key);
    store.dispatch(setActive(proj));
    store.dispatch(
      Layout.setProject({ slice: proj.layout as Layout.SliceState, keepNav: false }),
    );
  }, `Failed to select ${names}`);
};

const VALID_CHILDREN: ontology.ResourceType[] = [
  "schematic",
  "lineplot",
  "log",
  "table",
  "group",
];

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "project",
  icon: <Icon.Project />,
  onSelect: handleSelect,
  TreeContextMenu,
  canDrop: ({ items }) =>
    items.every(({ key }) => VALID_CHILDREN.some((c) => key.toString().includes(c))),
};
