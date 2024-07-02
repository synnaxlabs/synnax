// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu } from "@synnaxlabs/pluto";

import { Menu } from "@/components/menu";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";

type DeviceLayoutCreator = (
  device: string,
  initial: Partial<Layout.State>,
) => Layout.Creator;

const ZERO_LAYOUT_STATES: Record<string, DeviceLayoutCreator> = {
  [NI.Device.CONFIGURE_LAYOUT_TYPE]: NI.Device.createConfigureLayout,
  [OPC.Device.CONFIGURE_LAYOUT_TYPE]: OPC.Device.createConfigureLayout,
};

export const handleSelect: Ontology.HandleSelect = () => {};

const handleConfigure = ({
  selection,
  client,
  placeLayout,
}: Ontology.TreeContextMenuProps): void => {
  if (selection.nodes.length === 0) return;
  const first = selection.resources[0];
  void (async () => {
    const d = await client.hardware.devices.retrieve(first.id.key);
    const baseLayout = ZERO_LAYOUT_STATES[`configure_${d.make}`];
    if (baseLayout == null) return;
    return placeLayout(baseLayout(d.key, {}));
  })();
};

const handleDelete = ({
  selection,
  addStatus,
  client,
}: Ontology.TreeContextMenuProps): void => {
  if (selection.nodes.length === 0) return;
  void (async () => {
    try {
      await client.hardware.devices.delete(selection.resources.map((r) => r.id.key));
    } catch (e) {
      addStatus({
        key: "delete-device",
        variant: "error",
        message: `Failed to delete devices: ${(e as Error).message}`,
      });
    }
  })();
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { client, selection } = props;
  if (selection.nodes.length === 0) return null;
  const singleResource = selection.nodes.length === 1;

  const handleSelect = (itemKey: string): void => {
    switch (itemKey) {
      case "configure":
        handleConfigure(props);
        break;
      case "delete":
        handleDelete(props);
        break;
      case "link": {
        Link.CopyToClipboard({
          clusterKey: client.key,
          addStatus: props.addStatus,
          resource: {
            type: "device",
            key: selection.resources[0].id.key,
          },
          name: selection.resources[0].name,
        });
        break;
      }
    }
  };

  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      {singleResource && (
        <PMenu.Item itemKey="configure" startIcon={<Icon.Hardware />}>
          Configure
        </PMenu.Item>
      )}
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      {singleResource && <Link.CopyMenuItem />}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "device",
  hasChildren: false,
  icon: <Icon.Device />,
  canDrop: () => false,
  onSelect: handleSelect,
  TreeContextMenu,
  haulItems: () => [],
  allowRename: () => false,
  onRename: undefined,
};
