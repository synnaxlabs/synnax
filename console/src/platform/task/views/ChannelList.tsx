// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Header as PHeader } from "@synnaxlabs/lyra/header";
import { type ReactElement, type ReactNode, useEffect } from "react";

import { Button } from "@/platform/button";
import { Empty } from "@/platform/empty";
import { type BindChannelsProps } from "@/platform/task/BindChannels";
import {
  ChannelList as Base,
  type ChannelListProps as BaseProps,
} from "@/platform/task/ChannelList";
import { useIsPreview } from "@/platform/task/Form";
import { type Channel } from "@/platform/task/types";

const Header = ({ title }: { title: string }) => (
  <PHeader.Header>
    <PHeader.Title weight={500} color={10}>
      {title}
    </PHeader.Title>
  </PHeader.Header>
);

const EMPTY_CONTENT = <Empty.Action message="No channels in task" />;

interface FooterProps {
  label: string;
  onAdd: () => void;
}

const Footer = ({ label, onAdd }: FooterProps) => {
  const isPreview = useIsPreview();
  if (isPreview) return null;
  return (
    <Button.CreateListItem size="small" onClick={onAdd}>
      {label}
    </Button.CreateListItem>
  );
};

export interface ChannelListProps<C extends Channel> extends Omit<
  BaseProps<C>,
  | "data"
  | "header"
  | "emptyContent"
  | "footer"
  | "path"
  | "remove"
  | "onDuplicate"
  | "resolve"
> {
  /** Names the list in its header; null when an enclosing frame such as Panes does. */
  title?: string | null;
  /** Fills the list while it holds nothing. */
  emptyContent?: ReactElement;
  /** Labels the add action pinned under the list. */
  addLabel?: string;
  /** Replaces the add action, for a list that adds through its own flow. */
  footer?: ReactNode;
  resolve: BindChannelsProps<C>["resolve"] | null;
  /** Creates what the add action adds, or returns null to add nothing. */
  createChannel?: (channels: C[]) => C | null;
  createChannels?: (channels: C[], keys: string[]) => C[];
  path?: string;
}

/**
 * Lists the channels of a task, or other items such as endpoints, at path. An add or a
 * duplicate selects what it adds.
 */
export const ChannelList = <C extends Channel>({
  createChannel,
  createChannels,
  path = "config.channels",
  title = "Channels",
  emptyContent = EMPTY_CONTENT,
  addLabel = "Add channel",
  footer,
  ...rest
}: ChannelListProps<C>) => {
  const ctx = Form.useContext();

  // Hotfix for an issue we have been having with the "path config.channels" issue.
  // Would like to remove this but it's just something that comes up consistently in
  // prod and it's more important to make sure that we don't break anything in prod then
  // have pretty code.
  useEffect(() => {
    if (path === "config.channels") {
      const cfg = ctx.get("config").value;
      if (
        typeof cfg !== "object" ||
        cfg == null ||
        !("channels" in cfg) ||
        !Array.isArray(cfg.channels)
      )
        ctx.set(path, []);
    }
  }, [path, ctx]);

  const { data, push, remove } = Form.useFieldList<C["key"], C>(path);
  const { onSelect } = rest;
  const handleAdd =
    createChannel == null
      ? undefined
      : () => {
          const channel = createChannel(ctx.get<C[]>(path).value);
          if (channel == null) return;
          push(channel);
          onSelect([channel.key]);
        };
  const handleDuplicate =
    createChannels == null
      ? undefined
      : (chs: C[], keys: string[]) => {
          const duplicated = createChannels(chs, keys);
          if (duplicated.length === 0) return;
          push(duplicated);
          onSelect(duplicated.map(({ key }) => key));
        };
  return (
    <Base
      {...rest}
      data={data}
      header={title == null ? null : <Header title={title} />}
      emptyContent={emptyContent}
      footer={
        footer ?? (handleAdd != null && <Footer label={addLabel} onAdd={handleAdd} />)
      }
      path={path}
      remove={remove}
      onDuplicate={handleDuplicate}
    />
  );
};
