// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/settings/Page.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { useCardBackground } from "@/platform/settings/background";

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
          CSS.BE("settings", "card"),
          flush && CSS.BEM("settings", "card", "flush"),
        )}
      >
        {children}
      </Flex.Box>
    </Flex.Box>
  );
};

export interface EntryProps extends PropsWithChildren {
  label: string;
  /** Shows under the label. Text renders in the description style. */
  description?: ReactNode;
}

/** One setting: its label and description on the left, and its control on the right. */
export const Entry = ({ label, description, children }: EntryProps): ReactElement => (
  <Flex.Box
    x
    align="center"
    justify="between"
    gap="huge"
    className={CSS.BE("settings", "entry")}
  >
    <Flex.Box y gap="tiny" className={CSS.BE("settings", "entry-text")}>
      <Text.Text color={11}>{label}</Text.Text>
      {typeof description === "string" ? (
        <Text.Text level="small" color={9}>
          {description}
        </Text.Text>
      ) : (
        description
      )}
    </Flex.Box>
    {children}
  </Flex.Box>
);

/** Holds content that is not a list of entries, such as a form, inside a card. */
export const Body = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box y className={CSS.BE("settings", "body")}>
    {children}
  </Flex.Box>
);
