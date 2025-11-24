// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Toolbar } from "@/components";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";

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

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "device",
  icon: <Icon.Device />,
  content: <Content />,
  tooltip: "Devices",
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
  trigger: ["D"],
  useVisible: Device.useViewAccessGranted,
};
