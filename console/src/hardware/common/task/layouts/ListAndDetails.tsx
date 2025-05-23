// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Divider,
  Form,
  Header,
  type RenderProp,
} from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { CSS } from "@/css";
import {
  ChannelList,
  type ChannelListProps,
} from "@/hardware/common/task/layouts/ChannelList";
import { type Channel } from "@/hardware/common/task/types";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";

export interface CreateChannel<C extends Channel> {
  (channels: C[], index: number): C | null;
}

export interface DetailsProps {
  path: string;
}

export interface ListAndDetailsProps<C extends Channel>
  extends Pick<
    ChannelListProps<C>,
    "onTare" | "allowTare" | "isSnapshot" | "listItem" | "contextMenuItems"
  > {
  details: RenderProp<DetailsProps>;
  createChannel: CreateChannel<C>;
  initialChannels: C[];
}

export const ListAndDetails = <C extends Channel>({
  details,
  initialChannels,
  createChannel,
  ...rest
}: ListAndDetailsProps<C>) => {
  const [selected, setSelected] = useState<string[]>(
    initialChannels.length ? [initialChannels[0].key] : [],
  );
  const [selectedIndex, setSelectedIndex] = useState<number>(
    initialChannels.length ? 0 : -1,
  );
  const { get } = Form.useContext();
  const handleSelect = useCallback(
    (keys: string[], index: number) => {
      setSelected(keys);
      setSelectedIndex(index);
    },
    [setSelected, setSelectedIndex],
  );
  const handleCreateChannel = useCallback(
    (channels: C[]) => createChannel(channels, selectedIndex),
    [selectedIndex],
  );
  const handleDuplicateChannels = useCallback(
    (allChannels: C[], indices: number[]) => {
      const newlyMade: C[] = [];
      indices.forEach((index) => {
        const newlyMadeChannel = createChannel([...allChannels, ...newlyMade], index);
        if (newlyMadeChannel != null) newlyMade.push(newlyMadeChannel);
      });
      return newlyMade;
    },
    [createChannel],
  );
  const copy = useCopyToClipboard();
  const handleCopyChannelDetails = useCallback(() => {
    if (selectedIndex === -1) return;
    copy(
      binary.JSON_CODEC.encodeString(get(`config.channels.${selectedIndex}`).value),
      "channel details",
    );
  }, [selectedIndex, copy, get]);
  return (
    <>
      <ChannelList<C>
        {...rest}
        selected={selected}
        onSelect={handleSelect}
        createChannel={handleCreateChannel}
        createChannels={handleDuplicateChannels}
      />
      <Divider.Divider y />
      <Align.Space y grow empty className={CSS.B("details")}>
        <Header.Header level="p">
          <Header.Title weight={500} wrap={false} shade={10}>
            Details
          </Header.Title>
          <Header.Actions>
            <Button.Icon
              disabled={selectedIndex === -1}
              tooltip="Copy channel details as JSON"
              tooltipLocation="left"
              variant="text"
              onClick={handleCopyChannelDetails}
              shade={2}
            >
              <Icon.JSON style={{ color: "var(--pluto-gray-l9)" }} />
            </Button.Icon>
          </Header.Actions>
        </Header.Header>
        {selectedIndex === -1 ? null : (
          <Align.Space y className={CSS.BE("details", "form")} empty grow>
            {details({ path: `config.channels.${selectedIndex}` })}
          </Align.Space>
        )}
      </Align.Space>
    </>
  );
};
