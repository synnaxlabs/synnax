// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Access, Device, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type Nav } from "@/platform/nav";
import { Ontology } from "@/platform/ontology";
import { Toolbar } from "@/platform/toolbar";

const Content = (): ReactElement => {
  const { data: groupID } = Device.useRetrieveGroupID({});
  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Device />}>Devices</Toolbar.Title>
      </Toolbar.Header>
      <Ontology.Tree root={groupID} />
    </Toolbar.Content>
  );
};

export const TOOLBAR: Nav.Item = {
  key: "device",
  icon: <Icon.Device />,
  content: <Content />,
  tooltip: "Devices",
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
  trigger: ["D"],
  useVisible: () => Access.useRetrieveGranted(device.TYPE_ONTOLOGY_ID),
};
