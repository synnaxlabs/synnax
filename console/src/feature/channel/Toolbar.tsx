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
import { type ReactElement } from "react";

import { Channel as PlatformChannel } from "@/platform/channel";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Range } from "@/platform/range";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

const channelID = ({ key }: channel.Channel) => channel.ontologyID(key);

// Channels match by alias under the active range, as the tree shows them.
const useChannelMatches: Tree.UseMatches = (query) => {
  const folders = Tree.useOntologyMatches(query);
  const activeRange = Range.useResolve();
  const channels = Channel.useResultList(
    query == null
      ? null
      : { searchTerm: query.term, rangeKey: activeRange?.key, limit: query.limit },
  );
  return [...folders, Tree.toMatches(channels, channelID)];
};

const useSearch = Tree.createSearch({
  noun: "channels",
  useMatches: useChannelMatches,
});

interface ActionsProps {
  search: Tree.Search;
}

const Actions = ({ search }: ActionsProps): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const openCalculated = PlatformChannel.useCalculatedModal();
  const hasCreatePermission = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
  return (
    <Toolbar.Actions>
      <Tree.SearchAction search={search} />
      {hasCreatePermission && (
        <>
          <Toolbar.Action
            onClick={() => openCalculated()}
            tooltip="Create calculated channel"
          >
            <Channel.CreateCalculatedIcon />
          </Toolbar.Action>
          <Toolbar.Action onClick={() => openCreate()} tooltip="Create channel">
            <Icon.Add />
          </Toolbar.Action>
        </>
      )}
    </Toolbar.Actions>
  );
};

const Content = (): ReactElement => {
  // Does not suspend, so the header stays painted while the body reads the group.
  const channels = Channel.useResultGroup({});
  const search = useSearch(
    channels.variant === "success" ? group.ontologyID(channels.data.key) : null,
  );
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

const Body = ({ search }: BodyProps): ReactElement => {
  const g = Channel.useGroup({});
  const root = group.ontologyID(g.key);
  return <Tree.Tree root={root} emptyContent={<EmptyContent />} search={search} />;
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
