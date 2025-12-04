// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, task } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Mosaic, Task as Core } from "@synnaxlabs/pluto";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
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
    state: { getResource, shape },
  } = props;
  const { ids, rootID } = selection;
  const resources = getResource(ids);
  const handleDelete = useDelete(props);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = Common.Task.useExport();
  const snap = useRangeSnapshot();
  const range = Range.useSelect();
  const group = Group.useCreateFromSelection();
  const rename = useRename(props);
  const onSelect = {
    delete: handleDelete,
    edit: () =>
      handleSelect({
        selection: resources,
        placeLayout: props.placeLayout,
        client,
        addStatus,
        store,
        handleError,
        removeLayout: props.removeLayout,
        services: props.services,
      }),
    rename,
    link: () => handleLink({ name: resources[0].name, ontologyID: resources[0].id }),
    export: () => handleExport(ids[0].key),
    rangeSnapshot: () =>
      snap({ tasks: resources.map(({ id: { key }, name }) => ({ key, name })) }),
    group: () => group(props),
  };
  const singleResource = ids.length === 1;
  const hasNoSnapshots = resources.every((r) => r.data?.snapshot === false);
  return (
    <PMenu.Menu level="small" gap="small" onChange={onSelect}>
      <Group.MenuItem ids={ids} shape={shape} rootID={rootID} />
      {hasNoSnapshots && range?.persisted === true && (
        <>
          <Range.SnapshotMenuItem key="snapshot" range={range} />
          <PMenu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <PMenu.Item itemKey="edit">
            <Icon.Edit />
            {`${resources[0].data?.snapshot ? "View" : "Edit"} configuration`}
          </PMenu.Item>
          <Menu.RenameItem />
          <Link.CopyMenuItem />
          <Export.MenuItem />
          <PMenu.Divider />
        </>
      )}
      <PMenu.Item itemKey="delete">
        <Icon.Delete />
        Delete
      </PMenu.Item>
      <PMenu.Divider />
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
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
  haulItems: ({ id }) => [
    { type: Mosaic.HAUL_CREATE_TYPE, key: ontology.idToString(id) },
  ],
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
