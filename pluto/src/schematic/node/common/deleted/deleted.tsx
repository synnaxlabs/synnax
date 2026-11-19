// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Badge as Base } from "@synnaxlabs/lyra/badge";
import { Text as BaseText } from "@synnaxlabs/lyra/text";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Grid } from "@/schematic/node/common/grid";
import { telem } from "@/telem/aether";

/** The text a symbol shows in place of a channel that the Core deleted. */
export const TEXT = telem.DELETED_STATUS.message;

/**
 * @returns true when the Core holds none of a key in keys, as after a delete of its
 * channel. The zero key is never deleted.
 */
export const use = (keys: channel.Key[]): boolean => {
  const configured = keys.filter((k) => !primitive.isZero(k));
  const { data } = Channel.useResultMultiple(
    configured.length > 0 ? { where: { key: { in: configured } } } : null,
  );
  return data != null && configured.some((k) => !data.some((ch) => ch.key === k));
};

/** Says that the Core deleted the channel, inside a readout that shows its value. */
export const Text = (): ReactElement => (
  <BaseText.Text level="small" color={9} overflow="fade">
    {TEXT}
  </BaseText.Text>
);

/** Marks a symbol whose channel the Core deleted, below the symbol. */
export const Item = Grid.createItem((): ReactElement => (
  <Grid.Item itemKey="deleted" location="bottom">
    <Base.Badge>{TEXT}</Base.Badge>
  </Grid.Item>
));
