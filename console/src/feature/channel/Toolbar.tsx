// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, group } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Channel } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Channel as PlatformChannel } from "@/platform/channel";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

const Content = (): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const openCalculated = PlatformChannel.useCalculatedModal();
  const hasCreatePermission = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
  const [searching, setSearching] = useState(false);
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        <Toolbar.Title>
          <Icon.Channel />
          Channels
        </Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.SearchAction value={searching} onChange={setSearching} />
          {hasCreatePermission && (
            <>
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
            </>
          )}
        </Toolbar.Actions>
      </Toolbar.Header>
      <Toolbar.Body>
        <Body search={searching} />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};

interface BodyProps {
  search: boolean;
}

const Body = ({ search }: BodyProps): ReactElement => {
  const g = Channel.useGroup({});
  const root = group.ontologyID(g.key);
  return <Tree.Tree root={root} search={search} emptyContent={<EmptyContent />} />;
};

const EmptyContent = (): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No channels"
      action={hasCreatePermission ? "Create channel" : undefined}
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
