// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Device } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useConnectModal } from "@/feature/device/Connect";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

const Content = (): ReactElement => {
  const openConnect = useConnectModal();
  const canCreate = Access.useCreateGranted(device.TYPE_ONTOLOGY_ID);
  const groupID = Device.useGroupID({});
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        <Toolbar.Title>
          <Icon.Device />
          Devices
        </Toolbar.Title>
        {canCreate && (
          <Toolbar.Actions>
            <Toolbar.Action tooltip="Connect device" onClick={() => openConnect()}>
              <Icon.Add />
            </Toolbar.Action>
          </Toolbar.Actions>
        )}
      </Toolbar.Header>
      <Toolbar.Body>
        <Tree.Tree
          root={groupID}
          emptyContent={
            <Empty.Action
              message="No devices"
              action={canCreate ? "Connect device" : undefined}
              onClick={() => openConnect()}
            />
          }
        />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "device",
  icon: <Icon.Device />,
  content: <Content />,
  tooltip: "Devices",
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
  trigger: ["D"],
  useVisible: () => Access.useRetrieveGranted(device.TYPE_ONTOLOGY_ID),
};
