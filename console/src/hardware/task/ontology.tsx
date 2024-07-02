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

const ZERO_LAYOUT_STATES: Record<string, Layout.State> = {
  [OPC.Task.configureReadLayout.type]: OPC.Task.configureReadLayout,
  [NI.Task.configureAnalogReadLayout.type]: NI.Task.configureAnalogReadLayout,
  [NI.Task.configureDigitalWriteLayout.type]: NI.Task.configureDigitalWriteLayout,
  [NI.Task.configureDigitalReadLayout.type]: NI.Task.configureDigitalReadLayout,
};

const handleSelect: Ontology.HandleSelect = ({
  selection,
  placeLayout,
  client,
  addStatus,
}) => {
  if (selection.length === 0) return;
  const task = selection[0].id;
  void (async () => {
    try {
      const t = await client.hardware.tasks.retrieve(task.key);
      const baseLayout = ZERO_LAYOUT_STATES[t.type];
      placeLayout({ ...baseLayout, key: selection[0].id.key });
    } catch (e) {
      addStatus({ variant: "error", message: (e as Error).message });
    }
  })();
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { store, selection, client, addStatus } = props;
  const { resources } = selection;

  const _handleSelect = (itemKey: string): void => {
    switch (itemKey) {
      case "delete":
        client.hardware.tasks
          .delete(resources.map(({ id }) => BigInt(id.key)))
          .catch((e: Error) => {
            addStatus({
              variant: "error",
              key: "deleteTaskError",
              message: e.message,
            });
          });
        break;
      case "edit":
        handleSelect({
          selection: resources,
          placeLayout: props.placeLayout,
          client,
          addStatus,
          store,
          removeLayout: props.removeLayout,
          services: props.services,
        });
        break;
      case "link": {
        Link.CopyToClipboard({
          clusterKey: client.key,
          resource: {
            type: "task",
            key: resources[0].id.key,
          },
          addStatus,
          name: resources[0].name,
        });
        break;
      }
    }
  };

  const singleResource = resources.length === 1;

  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={_handleSelect}>
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      {singleResource && <Link.CopyMenuItem />}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "task",
  hasChildren: false,
  icon: <Icon.Task />,
  canDrop: () => false,
  onSelect: handleSelect,
  TreeContextMenu,
  haulItems: () => [],
  allowRename: () => false,
  onRename: undefined,
};
