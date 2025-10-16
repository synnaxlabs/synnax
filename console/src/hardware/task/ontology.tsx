// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, task } from "@synnaxlabs/client";
import {
  ContextMenu as PContextMenu,
  Icon,
  Mosaic,
  Task as Core,
} from "@synnaxlabs/pluto";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Common } from "@/hardware/common";
import { type FormLayoutArgs } from "@/hardware/common/task/Form";
import { createLayout, retrieveAndPlaceLayout } from "@/hardware/task/layouts";
import { useRangeSnapshot } from "@/hardware/task/useRangeSnapshot";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";
import { Range } from "@/range";

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

const useDelete = createUseDelete({
  type: "Task",
  query: Core.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout }) => {
    removeLayout(...data);
    return data;
  },
});

export const useRename = createUseRename({
  query: Core.useRename,
  ontologyID: task.ontologyID,
  convertKey: String,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    const layout = Layout.selectByFilter(
      store.getState(),
      (l) => (l.args as FormLayoutArgs)?.taskKey === key,
    );
    if (layout != null) {
      store.dispatch(Layout.rename({ key: layout.key, name }));
      rollbacks.push(() => Layout.rename({ key: layout.key, name: oldName }));
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
    state: { getResource },
  } = props;
  const { ids } = selection;
  const resources = getResource(ids);
  const handleDelete = useDelete(props);
  const copyLink = Cluster.useCopyLinkToClipboard();
  const exportTask = Common.Task.useExport();
  const snap = useRangeSnapshot();
  const range = Range.useSelect();
  const rename = useRename(props);
  const handleExport = () => exportTask(ids[0].key);
  const singleResource = ids.length === 1;
  const handleEdit = () =>
    handleSelect({
      selection: resources,
      placeLayout: props.placeLayout,
      client,
      addStatus,
      store,
      handleError,
      removeLayout: props.removeLayout,
      services: props.services,
    });
  const hasNoSnapshots = resources.every((r) => r.data?.snapshot === false);
  const handleLink = () =>
    copyLink({ name: resources[0].name, ontologyID: resources[0].id });
  const handleRangeSnapshot = () =>
    snap({ tasks: resources.map(({ id: { key }, name }) => ({ key, name })) });
  return (
    <>
      <Group.ContextMenuItem {...props} />
      {hasNoSnapshots && range?.persisted === true && (
        <>
          <Range.SnapshotContextMenuItem range={range} onClick={handleRangeSnapshot} />
          <PContextMenu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <PContextMenu.Item onClick={handleEdit}>
            <Icon.Edit />
            {`${resources[0].data?.snapshot ? "View" : "Edit"} configuration`}
          </PContextMenu.Item>
          <ContextMenu.RenameItem onClick={rename} />
          <Link.CopyContextMenuItem onClick={handleLink} />
          <Export.ContextMenuItem onClick={handleExport} />
          <PContextMenu.Divider />
        </>
      )}
      <ContextMenu.DeleteItem onClick={handleDelete} />
      <PContextMenu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </>
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
    const task = await client.hardware.tasks.retrieve({ key: id.key });
    const layout = createLayout(task);
    placeLayout({ ...layout, tab: { mosaicKey: nodeKey, location } });
  }, "Failed to load task layout");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "task",
  icon: <Icon.Task />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [
    { type: Mosaic.HAUL_CREATE_TYPE, key: ontology.idToString(id) },
  ],
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
