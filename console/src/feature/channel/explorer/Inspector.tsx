// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/channel/explorer/Inspector.css";

import { type channel, resource } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, Resource } from "@synnaxlabs/pluto";
import { compare } from "@synnaxlabs/x";
import { type ReactElement, Suspense } from "react";

import { CSS } from "@/platform/css";
import { Tree } from "@/platform/tree";

const CLASS = CSS.B("channel-inspector");

/** Orders uses by name, then by key, so equal names keep one order. */
const USE_ORDER = compare.chain<resource.Resource>(
  compare.by((r) => r.name),
  compare.by((r) => r.id.key),
);

interface EntryProps {
  resource: resource.Resource;
}

/** A record that uses the channel. A click opens it, as a double click in a tree does. */
const Entry = ({ resource: r }: EntryProps): ReactElement => {
  const Item = Tree.useItems()[r.id.type] ?? Tree.DefaultItem;
  const typeName = Item.typeName;
  return (
    <Item.OnSelect>
      {(onSelect) => (
        <Button.Button
          variant="text"
          size="small"
          justify="start"
          full="x"
          className={CSS.BE("channel-inspector", "entry")}
          tooltip={typeName == null ? "Open" : `Open ${typeName.toLowerCase()}`}
          tooltipLocation="left"
          onClick={() => onSelect(r)}
        >
          {Item.icon(r)}
          <Text.Text level="small" overflow="fade" color={10}>
            {r.name}
          </Text.Text>
          {typeName != null && (
            <Text.Text
              level="small"
              color={9}
              className={CSS.BE("channel-inspector", "type")}
            >
              {typeName}
            </Text.Text>
          )}
        </Button.Button>
      )}
    </Item.OnSelect>
  );
};

interface SectionProps {
  title: string;
  ids: resource.ID[];
}

const Resolved = ({ ids }: Pick<SectionProps, "ids">): ReactElement => {
  // A record can be deleted between the read of the uses and this read.
  const resources = Resource.useMultiple({ ids });
  return (
    <>
      {resources.toSorted(USE_ORDER).map((r) => (
        <Entry key={resource.idToString(r.id)} resource={r} />
      ))}
    </>
  );
};

/** The records that use the channel in one direction, under a heading with a count. */
const Section = ({ title, ids }: SectionProps): ReactElement | null => {
  if (ids.length === 0) return null;
  return (
    <Flex.Box y empty className={CSS.BE("channel-inspector", "section")}>
      <Text.Text level="small" weight={500} color={9}>
        {title} · {ids.length}
      </Text.Text>
      <Suspense fallback={null}>
        <Resolved ids={ids} />
      </Suspense>
    </Flex.Box>
  );
};

interface UsesProps {
  channelKey: channel.Key;
}

const Uses = ({ channelKey }: UsesProps): ReactElement | null => {
  const result = Channel.useResultMultiple({
    where: { key: { eq: channelKey } },
    includeUses: true,
  });
  if (result.variant === "error")
    return (
      <Status.Summary
        variant="error"
        message="Failed to load what uses the channel"
        description={result.status.description}
      />
    );
  const ch = result.data?.[0];
  if (ch == null) return null;
  const uses = ch.uses ?? [];
  if (uses.length === 0)
    return (
      <Text.Text level="small" color={9}>
        Nothing uses this channel.
      </Text.Text>
    );
  const of = (direction: channel.Direction) =>
    uses.filter((u) => u.direction === direction).map(({ id }) => id);
  return (
    <>
      <Section title="Written by" ids={of("write")} />
      <Section title="Read by" ids={of("read")} />
    </>
  );
};

export interface InspectorProps {
  /** The keys of the selected channels. */
  keys: channel.Key[];
}

/** Shows what writes and what reads the one selected channel. */
export const Inspector = ({ keys }: InspectorProps): ReactElement => {
  let content: ReactElement | null;
  if (keys.length === 0)
    content = (
      <Text.Text level="small" color={9}>
        Select a channel to see what uses it.
      </Text.Text>
    );
  else if (keys.length > 1)
    content = (
      <Text.Text level="small" color={9}>
        {keys.length} channels selected
      </Text.Text>
    );
  else content = <Uses key={keys[0]} channelKey={keys[0]} />;
  return (
    <Flex.Box y className={CLASS} aria-label="Channel inspector">
      {content}
    </Flex.Box>
  );
};
