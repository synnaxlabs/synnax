// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Icon, Tag, Text } from "@synnaxlabs/pluto";
import { type ReactNode } from "react";

export interface ItemLabelProps extends Pick<Text.TextProps, "color"> {
  /** What sort of item it is: a channel type, an HTTP method. Shown as a tag. */
  kind: string;
  icon?: Icon.ReactElement;
  /** The item's identity: a port, a path. */
  children: ReactNode;
}

/**
 * Names a task item in a list row or a details header: a tag for its kind, then its
 * identity. Both sites use it so the header echoes the row the user clicked.
 */
export const ItemLabel = ({ kind, icon, color = 10, children }: ItemLabelProps) => (
  <>
    <Tag.Tag size="small" icon={icon}>
      {kind}
    </Tag.Tag>
    <Text.Text level="p" weight={500} color={color} overflow="ellipsis">
      {children}
    </Text.Text>
  </>
);
