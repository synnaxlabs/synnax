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

export interface ItemLabelProps extends Pick<Text.TextProps, "color" | "variant"> {
  /** What sort of item it is: a channel type, an HTTP method. Shown as a tag. */
  kind: string;
  /**
   * Renders the kind as bare colored text instead of a tag: the hue marks the
   * category, so the chassis would be redundant. For closed vocabularies with
   * a stable color per value, e.g. HTTP methods.
   */
  kindColor?: Text.TextProps["color"];
  icon?: Icon.ReactElement;
  /** The item's identity: a port, a path. */
  children: ReactNode;
}

/* Colored kinds share a fixed column so every identity starts flush. */
const KIND_STYLE = { width: "5ch" };

/**
 * Names a task item in a list row or a details header: a tag for its kind, then its
 * identity. Both sites use it so the header echoes the row the user clicked.
 */
export const ItemLabel = ({
  kind,
  kindColor,
  icon,
  color = 10,
  variant,
  children,
}: ItemLabelProps) => (
  <>
    {kindColor == null ? (
      <Tag.Tag size="small" icon={icon}>
        {kind}
      </Tag.Tag>
    ) : (
      <Text.Text level="small" weight={600} color={kindColor} style={KIND_STYLE}>
        {icon}
        {kind}
      </Text.Text>
    )}
    <Text.Text
      level="p"
      weight={500}
      color={color}
      variant={variant}
      overflow="ellipsis"
    >
      {children}
    </Text.Text>
  </>
);
