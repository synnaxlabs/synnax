// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Button, Divider, Form, Header } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { type Channel } from "@/hardware/common/task/ChannelList";
import {
  DefaultChannelList,
  type DefaultChannelListProps,
} from "@/hardware/common/task/DefaultChannelList";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";

export interface GenerateChannelFn<C extends Channel> {
  (channels: C[], index: number): C | null;
}

export interface DetailsProps {
  path: string;
}

export interface ListAndDetailsProps<C extends Channel> {
  listItem: DefaultChannelListProps<C>["children"];
  details: (props: DetailsProps) => ReactElement;
  generateChannel: GenerateChannelFn<C>;
  isSnapshot: boolean;
  initalChannels: C[];
  onTare?: DefaultChannelListProps<C>["onTare"];
  allowTare?: DefaultChannelListProps<C>["allowTare"];
}

export const ListAndDetails = <C extends Channel>({
  listItem,
  details,
  initalChannels,
  generateChannel,
  ...props
}: ListAndDetailsProps<C>): ReactElement => {
  const [selected, setSelected] = useState<string[]>(
    initalChannels.length ? [initalChannels[0].key] : [],
  );
  const [selectedIndex, setSelectedIndex] = useState<number>(
    initalChannels.length ? 0 : -1,
  );
  const { get } = Form.useContext();
  const handleSelect = useCallback(
    (keys: string[], index: number) => {
      setSelected(keys);
      setSelectedIndex(index);
    },
    [setSelected, setSelectedIndex],
  );
  const handleGenerateChannel = useCallback(
    (channels: C[]) => generateChannel(channels, selectedIndex),
    [selectedIndex],
  );
  const copy = useCopyToClipboard();
  const handleCopyChannelDetails = useCallback(() => {
    if (selectedIndex === -1) return;
    copy(
      binary.JSON_CODEC.encodeString(get(`config.channels.${selectedIndex}`).value),
      "Channel details",
    );
  }, [selectedIndex, copy, get]);
  return (
    <>
      <DefaultChannelList<C>
        {...props}
        selected={selected}
        onSelect={handleSelect}
        generateChannel={handleGenerateChannel}
        style={{ height: "calc(100% - 18px)", width: "30rem" }}
      >
        {(p) => listItem(p)}
      </DefaultChannelList>
      <Divider.Divider direction="y" />
      <Align.Space direction="y" grow>
        <Header.Header level="h4">
          <Header.Title weight={500} wrap={false}>
            Details
          </Header.Title>
          <Header.Actions>
            <Button.Icon
              disabled={selectedIndex === -1}
              tooltip="Copy channel details as JSON"
              tooltipLocation="left"
              variant="text"
              onClick={handleCopyChannelDetails}
            >
              <Icon.JSON style={{ color: "var(--pluto-gray-l7)" }} />
            </Button.Icon>
          </Header.Actions>
        </Header.Header>
        <Align.Space className={CSS.B("details")}>
          {selectedIndex === -1 ? null : (
            <Align.Space
              direction="y"
              className={CSS.B("channel-form-content")}
              empty
              style={{ padding: "1rem 5rem", overflow: "scroll" }}
            >
              {details({ path: `config.channels.${selectedIndex}` })}
            </Align.Space>
          )}
        </Align.Space>
      </Align.Space>
    </>
  );
};
