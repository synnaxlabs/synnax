// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Component, Divider, Flex } from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

import { CSS } from "@/css";
import {
  ChannelList,
  type ChannelListProps,
} from "@/hardware/common/task/layouts/ChannelList";
import { DetailsHeader } from "@/hardware/common/task/layouts/DetailsHeader";
import { type Channel } from "@/hardware/common/task/types";

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
  const detailsPath = selected.length > 0 ? `config.channels.${selected[0]}` : null;
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
        <DetailsHeader path={detailsPath ?? ""} disabled={detailsPath == null} />
        {detailsPath != null && (
          <Flex.Box y className={CSS.BE("details", "form")} empty grow>
            {details({ path: detailsPath })}
          </Flex.Box>
        )}
      </Flex.Box>
    </>
  );
};
