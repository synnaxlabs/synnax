// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Nebula } from "@synnaxlabs/lyra/nebula";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Logo } from "@synnaxlabs/media";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { Root } from "@/ui/Root";

export interface CardProps extends PropsWithChildren {
  /** icon sits above the title, for a page that reports a state. */
  icon?: ReactElement;
  title: string;
  description?: ReactNode;
  error?: string | null;
  footer?: ReactNode;
}

/** Card is the frame of every auth page: title, form, and the links below it. */
export const Card = ({
  icon,
  title,
  description,
  error,
  footer,
  children,
}: CardProps): ReactElement => (
  <Root>
    <Nebula.Nebula />
    <nav className="portal-auth__islands">
      <a
        className="portal-auth__island portal-auth__island--square portal-frost"
        href="https://www.synnaxlabs.com"
        aria-label="Synnax"
      >
        <Logo variant="icon" />
      </a>
      <span className="portal-auth__island portal-frost">
        <Button.Button variant="text" textColor={10} href="https://docs.synnaxlabs.com">
          Docs
          <Icon.OpenExternal />
        </Button.Button>
      </span>
    </nav>
    <div className="portal-auth__stage">
      <Flex.Box y gap="huge" className="portal-auth portal portal-frost">
        <Flex.Box y gap="small" align="center">
          {icon}
          <Text.Text level="h3" weight={500}>
            {title}
          </Text.Text>
          {description != null && (
            <Text.Text level="p" color={9} className="portal-auth__description">
              {description}
            </Text.Text>
          )}
        </Flex.Box>
        <Flex.Box y gap="medium">
          {error != null && (
            <Status.Summary
              variant="error"
              level="small"
              message={error}
              className="portal-auth__error"
            />
          )}
          {children}
        </Flex.Box>
        {footer != null && (
          <Flex.Box y gap="small" align="center">
            {footer}
          </Flex.Box>
        )}
      </Flex.Box>
    </div>
  </Root>
);
