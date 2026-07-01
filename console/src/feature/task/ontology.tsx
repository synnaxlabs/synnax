// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, task } from "@synnaxlabs/client";
import { Access, Icon, Menu, Mosaic, Task as Base } from "@synnaxlabs/pluto";
import { useMemo } from "react";

import { Group } from "@/feature/group";
import { Link } from "@/feature/link";
import { Ontology } from "@/feature/ontology";
import { Range } from "@/feature/range";
import { useExport } from "@/feature/task/export";
import { createLayout, retrieveAndPlaceLayout } from "@/feature/task/layouts";
import { useRangeSnapshot } from "@/feature/task/useRangeSnapshot";
import { Cluster } from "@/primitive/cluster";
import { ContextMenu } from "@/primitive/context-menu";
import { Export } from "@/primitive/export";
import { type FormLayoutArgs } from "@/primitive/task/Form";
import { Session } from "@/session";

const handleSelect: Ontology.HandleSelect = ({
  selection,
  placeLayout,
  client,
  handleError,
}) => {
  if (selection.length === 0) return;
  const key = selection[0].id.key;
  const name = selection[0].name;
  handleError(
    async () => await retrieveAndPlaceLayout(client, key, placeLayout),
    `Could not open ${name}`,
  );
};

const useDelete = Ontology.createUseDelete({
  type: "Task",
  query: Base.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout }) => {
    removeLayout(...data);
    return data;
  },
});

export const useRename = Ontology.createUseRename({
  query: Base.useRename,
  ontologyID: task.ontologyID,
  convertKey: String,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    const layout = Session.Layout.selectByFilter(
      store.getState(),
      (l) => (l.args as FormLayoutArgs)?.taskKey === key,
    );
    if (layout != null) {
      store.dispatch(Session.Layout.rename({ key: layout.key, name }));
      rollbacks.push(() => Session.Layout.rename({ key: layout.key, name: oldName }));
    }
    return { ...data, name };
  },
});

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    store,
    selection,
    client,
    addStatus,
    handleError,
    state: { getResource, shape },
  } = props;
  const { ids, rootID } = selection;
  const resources = getResource(ids);
  const handleDelete = useDelete(props);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = useExport();
  const snap = useRangeSnapshot();
  const range = Session.Range.useSelectState();
  const group = Group.useCreateFromSelection();
  const rename = useRename(props);
  const ontologyIDs = useMemo(() => ids.map((id) => task.ontologyID(id.key)), [ids]);
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  const hasDeletePermission = Access.useDeleteGranted(ontologyIDs);
  const hasUpdatePermission = Access.useUpdateGranted(ontologyIDs);
  const handleEdit = () =>
    handleSelect({
      ...props,
      selection: resources,
      client,
      addStatus,
      store,
      handleError,
    });
  const singleResource = ids.length === 1;
  const hasNoSnapshots = resources.every((r) => r.data?.snapshot === false);
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          <Group.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            onClick={() => group(props)}
          />
          {singleResource && (
            <>
              <ContextMenu.RenameItem onClick={rename} />
              <Menu.Divider />
            </>
          )}
        </>
      )}
      {hasCreatePermission && hasNoSnapshots && range?.persisted === true && (
        <>
          <Range.SnapshotMenuItem
            key="snapshot"
            range={range}
            onClick={() =>
              snap({
                tasks: resources.map(({ id: { key }, name }) => ({ key, name })),
              })
            }
          />
          <Menu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <Menu.Item itemKey="edit" onClick={handleEdit}>
            <Icon.Edit />
            {`${resources[0].data?.snapshot ? "View" : "Edit"} configuration`}
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <Link.CopyContextMenuItem
            onClick={() =>
              handleLink({ name: resources[0].name, ontologyID: resources[0].id })
            }
          />
          <Export.ContextMenuItem onClick={() => handleExport(ids[0].key)} />
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && (
        <>
          <ContextMenu.DeleteItem onClick={handleDelete} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  placeLayout,
  nodeKey,
  location,
  handleError,
}) =>
  handleError(async () => {
    const task = await client.tasks.retrieve({ key: id.key });
    const layout = createLayout(task);
    placeLayout({ ...layout, tab: { mosaicKey: nodeKey, location } });
  }, "Failed to load task layout");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "task",
  icon: <Icon.Task />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
