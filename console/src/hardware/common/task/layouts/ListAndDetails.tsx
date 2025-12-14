// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  type Component,
  Divider,
  Flex,
  Form,
  Header,
  Icon,
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
  (channels: C[], channelKeyToCopy?: string): C | null;
}

export interface DetailsProps {
  path: string;
}

export interface ListAndDetailsProps<C extends Channel> extends Pick<
  ChannelListProps<C>,
  "onTare" | "allowTare" | "listItem" | "contextMenuItems"
> {
  details: Component.RenderProp<DetailsProps>;
  createChannel: CreateChannel<C>;
}

export const ListAndDetails = <C extends Channel>({
  details,
  createChannel,
  ...rest
}: ListAndDetailsProps<C>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const { get } = Form.useContext();
  const handleCreateChannel = useCallback(
    (channels: C[]) => createChannel(channels, selected[0]),
    [createChannel, selected],
  );
  const handleDuplicateChannels = useCallback(
    (allChannels: C[], keys: string[]) => {
      const newlyMade: C[] = [];
      keys.forEach((key) => {
        const newlyMadeChannel = createChannel([...allChannels, ...newlyMade], key);
        if (newlyMadeChannel != null) newlyMade.push(newlyMadeChannel);
      });
      return newlyMade;
    },
    [createChannel],
  );
  const copy = useCopyToClipboard();
  const handleCopyChannelDetails = useCallback(() => {
    if (selected.length === 0) return;
    copy(
      binary.JSON_CODEC.encodeString(get(`config.channels.${selected[0]}`).value),
      "channel details",
    );
  }, [copy, get, selected]);
  return (
    <>
      <ChannelList<C>
        {...rest}
        selected={selected}
        onSelect={setSelected}
        createChannel={handleCreateChannel}
        createChannels={handleDuplicateChannels}
      />
      <Divider.Divider y />
      <Flex.Box y grow empty className={CSS.B("details")}>
        <Header.Header>
          <Header.Title weight={500} wrap={false} color={10}>
            Details
          </Header.Title>
          <Header.Actions>
            <Button.Button
              disabled={selected.length === 0}
              tooltip="Copy channel details as JSON"
              tooltipLocation="left"
              variant="text"
              onClick={handleCopyChannelDetails}
              contrast={2}
            >
              <Icon.JSON style={{ color: "var(--pluto-gray-l9)" }} />
            </Button.Button>
          </Header.Actions>
        </Header.Header>
        {selected.length > 0 && (
          <Flex.Box y className={CSS.BE("details", "form")} empty grow>
            {details({ path: `config.channels.${selected[0]}` })}
          </Flex.Box>
        )}
      </Flex.Box>
    </>
  );
};
