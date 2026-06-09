// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";

import { LinePlot } from "@/lineplot";
import { type Link } from "@/link";
import { Range } from "@/range";
import { Workspace } from "@/workspace";

export const handleLink: Link.Handler = async ({ client, key, placeLayout, store }) => {
  const channel = await client.channels.retrieve(key);
  const workspace = Workspace.selectActiveKey(store.getState()) ?? uuid.ZERO;
  const activeRange = Range.selectActiveKey(store.getState()) ?? Range.RECENT_RANGE_KEY;
  const { key: plotKey, name } = await client.lineplots.create(workspace, {
    name: `${channel.name} Plot`,
    channels: { y1: [channel.key] },
    ranges: { x1: [activeRange] },
  });
  placeLayout(LinePlot.create({ key: plotKey, name }));
};
