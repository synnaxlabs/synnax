// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import {
  ChannelList,
  type ChannelListProps,
} from "@/hardware/common/task/layouts/ChannelList";
import { type Channel } from "@/hardware/common/task/types";

export interface ListProps<C extends Channel> extends Pick<
  ChannelListProps<C>,
  "createChannel" | "listItem" | "contextMenuItems"
> {}

export const List = <C extends Channel>(props: ListProps<C>) => {
  const [selected, setSelected] = useState<string[]>([]);
  return <ChannelList<C> {...props} selected={selected} onSelect={setSelected} />;
};
