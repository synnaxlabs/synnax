// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useState } from "react";

import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { useIsPreview } from "@/platform/task/Form";
import { type Channel } from "@/platform/task/types";
import {
  ChannelList,
  type ChannelListProps,
  useAdd,
} from "@/platform/task/views/ChannelList";
import { Panes } from "@/platform/task/views/Panes";

export interface CreateChannel<C extends Channel> {
  (channels: C[], channelKeyToCopy?: string): C | null;
}

export interface DetailsProps {
  path: string;
}

export interface ListAndDetailsProps<C extends Channel> extends Pick<
  ChannelListProps<C>,
  "onTare" | "allowTare" | "listItem" | "contextMenuItems" | "resolve" | "path"
> {
  details: Component.RenderProp<DetailsProps>;
  /** Names the selected channel in the details header, typically by its port. */
  detailsTitle?: Component.RenderProp<DetailsProps>;
  createChannel: CreateChannel<C>;
}

export const ListAndDetails = <C extends Channel>({
  details,
  detailsTitle,
  createChannel,
  path = "config.channels",
  ...rest
}: ListAndDetailsProps<C>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const handleCreateChannel = (channels: C[]) => createChannel(channels, selected[0]);
  const handleDuplicateChannels = (allChannels: C[], keys: string[]) => {
    const newlyMade: C[] = [];
    keys.forEach((key) => {
      const newlyMadeChannel = createChannel([...allChannels, ...newlyMade], key);
      if (newlyMadeChannel != null) newlyMade.push(newlyMadeChannel);
    });
    return newlyMade;
  };
  const handleAdd = useAdd({
    path,
    createChannel: handleCreateChannel,
    onSelect: setSelected,
  });
  const isPreview = useIsPreview();
  const detailsPath = selected.length > 0 ? `${path}.${selected[0]}` : null;
  return (
    <Panes
      listTitle="Channels"
      list={
        <ChannelList<C>
          {...rest}
          path={path}
          grow
          header={null}
          selected={selected}
          onSelect={setSelected}
          createChannel={handleCreateChannel}
          createChannels={handleDuplicateChannels}
        />
      }
      detailsPath={detailsPath}
      title={detailsPath != null && detailsTitle?.({ path: detailsPath })}
    >
      {detailsPath != null ? (
        <Flex.Box y className={CSS.BE("details", "form")} empty grow>
          {details({ path: detailsPath })}
        </Flex.Box>
      ) : (
        <Empty.Action
          grow
          message="No channel selected"
          action={isPreview ? undefined : "Add channel"}
          onClick={handleAdd}
        />
      )}
    </Panes>
  );
};
