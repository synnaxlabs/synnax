// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Rack, Status, Text, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components";
import { Group } from "@/group";
import { Sequence } from "@/hardware/task/sequence";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";

const CreateSequenceIcon = Icon.createComposite(Icon.Control, {
  topRight: Icon.Add,
});

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = Ontology.useConfirmDelete({ type: "Rack" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({
      state: { nodes, setNodes, getResource },
      selection: { resourceIDs },
    }) => {
      const resources = resourceIDs.map((id) => getResource(id));
      if (!(await confirm(resources))) throw new errors.Canceled();
      const prevNodes = Tree.deepCopy(nodes);
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => ontology.idToString(id)),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({ selection: { resourceIDs }, client }) =>
      await client.hardware.racks.delete(resourceIDs.map((id) => Number(id.key))),
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(e)) return;
      handleError(e, "Failed to delete racks");
    },
  }).mutate;
};

const useCopyKeyToClipboard = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const copy = useCopyToClipboard();
  return ({ selection: { resourceIDs }, state: { getResource } }) => {
    copy(resourceIDs[0].key, `key to ${getResource(resourceIDs[0]).name}`);
  };
};

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) => {
    const rack = await client.hardware.racks.retrieve({ key: Number(id.key) });
    await client.hardware.racks.create({ ...rack, name });
  },
};

const Item = ({ id, onRename, resource, ...rest }: Ontology.TreeItemProps) => {
  const { itemKey } = rest;
  const res = Rack.useRetrieve({ key: Number(id.key) });
  const status = res.data?.status;

  return (
    <Tree.Item {...rest}>
      <Icon.Rack />
      <Text.MaybeEditable
        id={itemKey}
        allowDoubleClick={false}
        value={resource.name}
        onChange={(name) => onRename?.(name)}
        overflow="ellipsis"
        style={{ width: 0, flexGrow: 1 }}
      />
      <Rack.StatusIndicator status={status} />
    </Tree.Item>
  );
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    state: { shape },
  } = props;
  const { resourceIDs, rootID } = selection;
  const handleDelete = useDelete();
  const placeLayout = Layout.usePlacer();
  const rename = Modals.useRename();
  const handleError = Status.useErrorHandler();
  const group = Group.useCreateFromSelection();
  const copyKeyToClipboard = useCopyKeyToClipboard();
  const createSequence = () => {
    handleError(async () => {
      const layout = await Sequence.createLayout({
        rename,
        rackKey: Number(resourceIDs[0].key),
      });
      if (layout == null) return;
      placeLayout(layout);
    }, "Failed to create control sequence");
  };
  const onSelect = {
    group: () => group(props),
    rename: () => Text.edit(ontology.idToString(resourceIDs[0])),
    createSequence,
    copy: () => copyKeyToClipboard(props),
    delete: () => handleDelete(props),
  };
  const isSingle = resourceIDs.length === 1;
  return (
    <PMenu.Menu level="small" gap="small" onChange={onSelect}>
      <Group.MenuItem
        resourceIDs={resourceIDs}
        rootID={rootID}
        shape={shape}
        showBottomDivider
      />
      {isSingle && (
        <>
          <Menu.RenameItem />
          <PMenu.Item itemKey="createSequence">
            <CreateSequenceIcon />
            Create Control Sequence
          </PMenu.Item>
          <PMenu.Item itemKey="copy">
            <Icon.Copy />
            Copy Key
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <Menu.DeleteItem />
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "rack",
  icon: <Icon.Rack />,
  hasChildren: true,
  canDrop: () => false,
  onSelect: () => {},
  haulItems: () => [],
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
  Item,
};
