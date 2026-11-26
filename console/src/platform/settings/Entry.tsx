// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/settings/Entry.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import type { PropsWithChildren, ReactElement, ReactNode } from "react";

import { CSS } from "@/platform/css";

export interface EntryProps extends PropsWithChildren {
  label: string;
  /** Shows under the label. Text renders in the description style. */
  description?: ReactNode;
}

/**
 * One setting: its label and description on the left, and its control on the right. The
 * control moves under the label when the two do not fit side by side.
 */
export const Entry = ({ label, description, children }: EntryProps): ReactElement => (
  <div className={CSS.BE("settings", "entry")}>
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
  </div>
);

/** Holds content that is not a list of entries, such as a form, inside a card. */
export const Body = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box y className={CSS.BE("settings", "body")}>
    {children}
  </Flex.Box>
);
