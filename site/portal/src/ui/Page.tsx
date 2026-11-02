// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { Panel } from "@/ui/Panel";
import { Root } from "@/ui/Root";
import { Tile } from "@/ui/Tile";

export interface Parent {
  label: string;
  href: string;
}

export interface PageProps extends PropsWithChildren {
  title: string;
  /** parent links the page that lists this one, above the title. */
  parent?: Parent;
  /** subtitle reads under the title: an organization name, a status line. */
  subtitle?: ReactNode;
  actions?: ReactNode;
  /** error replaces the page body with the message a load failed with. */
  error?: string | null;
}

/** Page is the frame every portal page renders: a header with actions, then content. */
export const Page = ({
  title,
  parent,
  subtitle,
  actions,
  error,
  children,
}: PageProps): ReactElement => (
  <Root>
    <Flex.Box y gap="huge" full="x">
      <Flex.Box x justify="between" align="end" gap="large" wrap>
        <Flex.Box y gap="tiny">
          {parent != null && (
            <Text.Text
              level="small"
              color={9}
              href={parent.href}
              className="portal-crumb"
            >
              <Icon.Caret.Left />
              {parent.label}
            </Text.Text>
          )}
          <Text.Text level="h2" color={11}>
            {title}
          </Text.Text>
          {subtitle != null && (
            <Text.Text level="p" color={9}>
              {subtitle}
            </Text.Text>
          )}
        </Flex.Box>
        {actions != null && (
          <Flex.Box x align="center" gap="small">
            {actions}
          </Flex.Box>
        )}
      </Flex.Box>
      {error != null ? (
        <Status.Summary variant="error" level="p" message={error} />
      ) : (
        children
      )}
    </Flex.Box>
  </Root>
);

export interface SectionProps extends PropsWithChildren {
  title: string;
  actions?: ReactNode;
}

export const Section = ({ title, actions, children }: SectionProps): ReactElement => (
  <Flex.Box y gap="medium" full="x">
    <Flex.Box x justify="between" align="center" gap="medium">
      <Text.Text level="h4" color={11}>
        {title}
      </Text.Text>
      {actions}
    </Flex.Box>
    {children}
  </Flex.Box>
);

export interface EmptyProps {
  icon?: ReactElement;
  message: string;
  description?: string;
  action?: ReactElement;
}

/** Empty is the body of a list with nothing in it. */
export const Empty = ({
  icon,
  message,
  description,
  action,
}: EmptyProps): ReactElement => (
  <Panel align="center" gap="small" className="portal-empty">
    {icon != null && <Tile>{icon}</Tile>}
    <Text.Text level="h5" color={11}>
      {message}
    </Text.Text>
    {description != null && (
      <Text.Text level="p" color={9} align="center">
        {description}
      </Text.Text>
    )}
    {action != null && <span className="portal-empty__action">{action}</span>}
  </Panel>
);
