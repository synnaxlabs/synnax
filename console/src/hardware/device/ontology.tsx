// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Text, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import {
  CONFIGURE_LAYOUTS,
  getContextMenuItems,
  getIcon,
  getMake,
  makeZ,
} from "@/hardware/device/make";
import { Ontology } from "@/ontology";

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) => {
    const device = await client.hardware.devices.retrieve(id.key);
    await client.hardware.devices.create({ ...device, name });
  },
};

const handleConfigure = ({
  selection: { resources },
  placeLayout,
  handleException,
}: Ontology.TreeContextMenuProps) => {
  const resource = resources[0];
  try {
    const make = makeZ.parse(resource.data?.make);
    placeLayout({ ...CONFIGURE_LAYOUTS[make], key: resource.id.key });
  } catch (e) {
    handleException(e, `Failed to configure ${resource.name}`);
  }
};

const useDelete = () => {
  const confirm = Ontology.useConfirmDelete({ type: "Device" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ state: { nodes, setNodes }, selection: { resources } }) => {
      const prevNodes = Tree.deepCopy(nodes);
      if (!(await confirm(resources))) throw errors.CANCELED;
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => id.toString()),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({ selection, client }) =>
      await client.hardware.devices.delete(selection.resources.map((r) => r.id.key)),
    onError: (e, { handleException, state: { setNodes } }, prevNodes) => {
      if (errors.CANCELED.matches(e)) return;
      if (prevNodes != null) setNodes(prevNodes);
      handleException(e, `Failed to delete devices`);
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { nodes, resources },
  } = props;
  const singleResource = nodes.length === 1;
  const first = resources[0];
  const handleDelete = useDelete();
  const group = Group.useCreateFromSelection();
  if (nodes.length === 0) return null;
  const handleSelect = {
    configure: () => handleConfigure(props),
    delete: () => handleDelete(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    group: () => group(props),
  };
  const C = singleResource ? getContextMenuItems(resources[0].data?.make) : null;
  const customMenuItems = C ? <C {...props} /> : null;
  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Group.MenuItem selection={selection} />
      {singleResource && (
        <>
          <Menu.RenameItem />
          {first.data?.configured !== true && (
            <>
              <PMenu.Divider />
              <PMenu.Item itemKey="configure" startIcon={<Icon.Hardware />}>
                Configure
              </PMenu.Item>
            </>
          )}
        </>
      )}
      <PMenu.Divider />
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      {customMenuItems != null && (
        <>
          <PMenu.Divider />
          {customMenuItems}
        </>
      )}
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const icon = (resource: ontology.Resource) => getIcon(getMake(resource.data?.make));

const Item: Tree.Item = ({ entry, ...rest }: Tree.ItemProps) => (
  <Tree.DefaultItem {...rest} entry={entry}>
    {({ entry, onRename, key }) => (
      <>
        <Text.MaybeEditable
          id={`text-${key}`}
          level="p"
          allowDoubleClick={false}
          value={entry.name}
          disabled={!entry.allowRename}
          onChange={(name) => onRename?.(entry.key, name)}
        />
        <Text.Text
          level="small"
          shade={6}
          style={{ lineHeight: "100%", marginTop: "0.25rem" }}
        >
          {entry.extraData?.location as string}
        </Text.Text>
      </>
    )}
  </Tree.DefaultItem>
);

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: device.ONTOLOGY_TYPE,
  icon,
  hasChildren: false,
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
  Item,
};
