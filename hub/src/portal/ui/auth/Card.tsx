// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Status, Text } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { Root } from "@/portal/ui/Root";

export interface CardProps extends PropsWithChildren {
  title: string;
  description?: ReactNode;
  error?: string | null;
  footer?: ReactNode;
}

/** Card is the frame of every auth page: title, form, and the links below it. */
export const Card = ({
  title,
  description,
  error,
  footer,
  children,
}: CardProps): ReactElement => (
  <Root>
    <Flex.Box y gap="large" className="portal-auth portal">
      <Flex.Box y gap="small">
        <Text.Text level="h2">{title}</Text.Text>
        {description != null && (
          <Text.Text level="p" color={9}>
            {description}
          </Text.Text>
        )}
      </Flex.Box>
      <Flex.Box
        y
        gap="medium"
        bordered
        rounded
        background={1}
        style={{ padding: "4rem" }}
      >
        {children}
        {error != null && (
          <Status.Summary variant="error" level="small" message={error} />
        )}
      </Flex.Box>
      {footer != null && (
        <Flex.Box y gap="small" align="center">
          {footer}
        </Flex.Box>
      )}
    </Flex.Box>
  </Root>
);
