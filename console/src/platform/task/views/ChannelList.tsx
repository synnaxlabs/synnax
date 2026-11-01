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
import { type ReactNode, useEffect } from "react";

import { Button } from "@/platform/button";
import { Empty } from "@/platform/empty";
import { type BindChannelsProps } from "@/platform/task/BindChannels";
import {
  ChannelList as Base,
  type ChannelListProps as BaseProps,
} from "@/platform/task/ChannelList";
import { type Channel } from "@/platform/task/types";
import { useIsPreview } from "@/platform/task/useIsPreview";

interface HeaderProps {
  actions?: ReactNode;
}

export const Header = ({ actions }: HeaderProps) => (
  <PHeader.Header>
    <PHeader.Title weight={500} color={10}>
      Channels
    </PHeader.Title>
    {actions != null && <PHeader.Actions>{actions}</PHeader.Actions>}
  </PHeader.Header>
);

const HEADER = <Header />;

const EMPTY_CONTENT = <Empty.Action message="No channels in task" />;

interface FooterProps {
  onAdd: () => void;
}

const Footer = ({ onAdd }: FooterProps) => {
  const isPreview = useIsPreview();
  if (isPreview) return null;
  return (
    <Button.CreateListItem size="small" onClick={onAdd}>
      Add channel
    </Button.CreateListItem>
  );
};

export interface ChannelListProps<C extends Channel> extends Omit<
  BaseProps<C>,
  "data" | "header" | "emptyContent" | "path" | "remove" | "onDuplicate" | "resolve"
> {
  /** Defaults to a "Channels" title; null when the enclosing frame titles the list. */
  header?: ReactNode;
  resolve: BindChannelsProps<C>["resolve"];
  createChannel: (channels: C[]) => C | null;
  createChannels?: (channels: C[], keys: string[]) => C[];
  path?: string;
  /** Shown in place of the items and the "Add channel" footer. */
  ports?: ReactNode;
}

export const ChannelList = <C extends Channel>({
  createChannel,
  createChannels,
  path = "config.channels",
  header = HEADER,
  ports,
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
  const handleAdd = () => {
    const channels = ctx.get<C[]>(path).value;
    const channel = createChannel(channels);
    if (channel == null) return;
    push(channel);
    onSelect([channel.key]);
  };
  const handleDuplicate = (chs: C[], keys: string[]) => {
    if (createChannels == null) return;
    const duplicated = createChannels(chs, keys);
    push(duplicated);
    return duplicated.map(({ key }) => key);
  };
  return (
    <Base
      {...rest}
      data={data}
      header={header}
      emptyContent={EMPTY_CONTENT}
      footer={ports == null && <Footer onAdd={handleAdd} />}
      ports={ports}
      path={path}
      remove={remove}
      onDuplicate={handleDuplicate}
    />
  );
};
