// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Channel } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { listTreeRoot } from "@/feature/channel/tree";
import { Channel as PlatformChannel } from "@/platform/channel";
import { Empty } from "@/platform/empty";
import { Group } from "@/platform/group";
import { type Nav } from "@/platform/nav";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

const toHit = (ch: channel.Channel) => Group.toHit("channel")(ch.payload);

const useChannelMatches: Tree.UseMatches = (query) => {
  const folders = Group.useMatches(query);
  const channels = Channel.useResultList(
    query == null ? null : { searchTerm: query.term, limit: query.limit },
  );
  return [...folders, Tree.toMatches(channels, toHit)];
};

const useSearch = Tree.createSearch({
  noun: "channels",
  useMatches: useChannelMatches,
  usePlacement: Group.usePlacement,
});

interface ActionsProps {
  search: Tree.Search;
}

const Actions = ({ search }: ActionsProps): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const openCalculated = PlatformChannel.useCalculatedModal();
  const hasCreatePermission = Access.useCreateGranted(channel.TYPE_RESOURCE_ID);
  return (
    <Toolbar.Actions>
      <Tree.SearchAction search={search} />
      {hasCreatePermission && (
        <>
          <Toolbar.CreateAction
            onClick={() => openCalculated()}
            tooltip="Create calculated channel"
          >
            <Channel.CreateCalculatedIcon />
          </Toolbar.CreateAction>
          <Toolbar.CreateAction onClick={() => openCreate()} tooltip="Create channel" />
        </>
      )}
    </Toolbar.Actions>
  );
};

const Content = (): ReactElement => {
  const search = useSearch(channel.TYPE_RESOURCE_ID);
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        {search.open ? (
          <Tree.SearchField search={search} />
        ) : (
          <>
            <Toolbar.Title>
              <Icon.Channel />
              Channels
            </Toolbar.Title>
            <Actions search={search} />
          </>
        )}
      </Toolbar.Header>
      <Toolbar.Body>
        <Body search={search} />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};

interface BodyProps {
  search: Tree.Search;
}

const Body = ({ search }: BodyProps): ReactElement => (
  <Tree.Tree
    root={channel.TYPE_RESOURCE_ID}
    listRoot={listTreeRoot}
    emptyContent={<EmptyContent />}
    search={search}
  />
);

const EmptyContent = (): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(channel.TYPE_RESOURCE_ID);
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
  useVisible: () => Access.useRetrieveGranted(channel.TYPE_RESOURCE_ID),
};
