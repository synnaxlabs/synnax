// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, group } from "@synnaxlabs/client";
import { Access, Channel, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Channel as PlatformChannel } from "@/platform/channel";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

const Actions = (): ReactElement | null => {
  const openCreate = PlatformChannel.useCreateModal();
  const openCalculated = PlatformChannel.useCalculatedModal();
  const hasCreatePermission = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
  if (!hasCreatePermission) return null;
  return (
    <Toolbar.Actions>
      <Toolbar.Action
        onClick={() => openCalculated()}
        tooltip="Create calculated channel"
        variant="filled"
      >
        <Channel.CreateCalculatedIcon />
      </Toolbar.Action>
      <Toolbar.Action
        onClick={() => openCreate()}
        tooltip="Create channel"
        variant="filled"
      >
        <Icon.Add />
      </Toolbar.Action>
    </Toolbar.Actions>
  );
};

const Content = (): ReactElement => {
  const { data: g } = Channel.useRetrieveGroup({});
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        <Toolbar.Title icon={<Icon.Channel />}>Channels</Toolbar.Title>
        <Actions />
      </Toolbar.Header>
      <Tree.Tree
        root={g == null ? undefined : group.ontologyID(g.key)}
        emptyContent={<EmptyContent />}
      />
    </Toolbar.Content>
  );
};

const EmptyContent = (): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No channels."
      action={hasCreatePermission ? "Create a channel" : undefined}
      onClick={() => openCreate()}
    />
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "channel",
  icon: <Icon.Channel />,
  content: <Content />,
  tooltip: "Channels",
  trigger: ["C"],
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
  useVisible: () => Access.useRetrieveGranted(channel.TYPE_ONTOLOGY_ID),
};
