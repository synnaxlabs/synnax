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

export interface PageProps extends PropsWithChildren {
  title: string;
}

/** Lays out one section of Settings: its title, then its groups. */
export const Page = ({ title, children }: PageProps): ReactElement => (
  <Flex.Box y className={CSS.BE("settings", "content")}>
    <Text.Text level="h3">{title}</Text.Text>
    {children}
  </Flex.Box>
);

export interface CardProps extends PropsWithChildren {
  title?: string;
}

/** Holds related entries in one card, under an optional title. */
export const Card = ({ title, children }: CardProps): ReactElement => (
  <Flex.Box y gap="medium">
    {title != null && <Text.Text level="h5">{title}</Text.Text>}
    <div className={CSS.BE("settings", "card")}>{children}</div>
  </Flex.Box>
);

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
