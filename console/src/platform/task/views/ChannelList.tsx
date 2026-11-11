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
import { type deep } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { Button } from "@/platform/button";
import { Empty } from "@/platform/empty";
import { Section } from "@/platform/section";
import { type BindChannelsProps } from "@/platform/task/BindChannels";
import {
  ChannelList as Base,
  type ChannelListProps as BaseProps,
} from "@/platform/task/ChannelList";
import { type Channel } from "@/platform/task/types";
import { useIsPreview } from "@/platform/task/useIsPreview";

interface HeaderProps {
  title?: string;
  actions?: ReactNode;
}

export const Header = ({ title = "Channels", actions }: HeaderProps) => (
  <Section.Header>
    <PHeader.Title weight={500} color={10}>
      {title}
    </PHeader.Title>
    {actions != null && <PHeader.Actions>{actions}</PHeader.Actions>}
  </Section.Header>
);

const HEADER = <Header />;

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

export interface UseAddProps<C extends Channel> {
  path: deep.Path<C[]>;
  createChannel?: (channels: C[]) => C | null;
  onSelect: (keys: string[]) => void;
  /** Skips selecting the new channel, since checkable lists select by checkbox. */
  checkable?: boolean;
}

/**
 * @returns a callback that appends a new channel to the list and selects it, or
 * undefined when there is no createChannel.
 */
export const useAdd = <C extends Channel>({
  path,
  createChannel,
  onSelect,
  checkable = false,
}: UseAddProps<C>): (() => void) | undefined => {
  const ctx = Form.useContext();
  const { push } = Form.useFieldList(path);
  if (createChannel == null) return undefined;
  return () => {
    const channel = createChannel(ctx.get(path).value);
    if (channel == null) return;
    push(channel);
    if (!checkable) onSelect([channel.key]);
  };
};

export interface ChannelListProps<C extends Channel> extends Omit<
  BaseProps<C>,
  "data" | "header" | "emptyContent" | "footer" | "remove" | "onDuplicate" | "resolve"
> {
  /** Defaults to a "Channels" title; null when the enclosing frame titles the list. */
  header?: ReactNode;
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
  /** Shown in place of the items and the footer. */
  ports?: ReactNode;
}

/**
 * Lists the channels of a task, or other items such as endpoints, at path. An add or a
 * duplicate selects what it adds.
 */
export const ChannelList = <C extends Channel>({
  createChannel,
  createChannels,
  path,
  header = HEADER,
  emptyContent = EMPTY_CONTENT,
  addLabel = "Add channel",
  footer,
  ports,
  ...rest
}: ChannelListProps<C>) => {
  const { data, push, remove } = Form.useFieldList(path);
  const { onSelect, checkable = false } = rest;
  const handleAdd = useAdd({ path, createChannel, onSelect, checkable });
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
      header={header}
      emptyContent={emptyContent}
      footer={
        ports == null &&
        (footer ?? (handleAdd != null && <Footer label={addLabel} onAdd={handleAdd} />))
      }
      ports={ports}
      path={path}
      remove={remove}
      onDuplicate={handleDuplicate}
    />
  );
};
