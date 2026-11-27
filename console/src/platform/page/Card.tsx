// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/page/Card.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import type { PropsWithChildren, ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useCardBackground } from "@/platform/page/background";

export interface CardProps extends PropsWithChildren {
  title?: string;
  /** Drops the card's inner padding, for content that pads itself, such as a tree. */
  flush?: boolean;
}

/** Holds related entries in one card, under an optional title. */
export const Card = ({ title, flush = false, children }: CardProps): ReactElement => {
  const background = useCardBackground();
  return (
    <Flex.Box y gap="medium">
      {title != null && <Text.Text level="h5">{title}</Text.Text>}
      <Flex.Box
        y
        empty
        bordered
        rounded="medium"
        background={background}
        className={CSS.cls(
          CSS.BE("page", "card"),
          flush && CSS.BEM("page", "card", "flush"),
        )}
      >
        {children}
      </Flex.Box>
    </Flex.Box>
  );
};
