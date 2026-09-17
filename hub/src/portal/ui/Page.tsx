// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Header, Status, Text } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { Root } from "@/portal/ui/Root";

export interface PageProps extends PropsWithChildren {
  title: string;
  /** subtitle reads under the title: an organization name, a status line. */
  subtitle?: ReactNode;
  actions?: ReactNode;
  /** error replaces the page body with the message a load failed with. */
  error?: string | null;
}

/** Page is the frame every portal page renders: a header with actions, then content. */
export const Page = ({
  title,
  subtitle,
  actions,
  error,
  children,
}: PageProps): ReactElement => (
  <Root>
    <Flex.Box y gap="huge" full="x">
      <Header.Header level="h2" bordered={false}>
        <Flex.Box y gap="tiny" grow>
          <Header.Title level="h2">{title}</Header.Title>
          {subtitle != null && (
            <Text.Text level="p" color={9}>
              {subtitle}
            </Text.Text>
          )}
        </Flex.Box>
        {actions != null && <Header.Actions>{actions}</Header.Actions>}
      </Header.Header>
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

/** Section is a titled block within a page. */
export const Section = ({ title, actions, children }: SectionProps): ReactElement => (
  <Flex.Box y gap="medium" full="x">
    <Header.Header level="h4" bordered={false}>
      <Header.Title level="h4">{title}</Header.Title>
      {actions != null && <Header.Actions>{actions}</Header.Actions>}
    </Header.Header>
    {children}
  </Flex.Box>
);

export interface EmptyProps {
  message: string;
  description?: string;
}

/** Empty is the body of a list with nothing in it. */
export const Empty = ({ message, description }: EmptyProps): ReactElement => (
  <Flex.Box y align="center" gap="small" style={{ padding: "6rem 0" }}>
    <Text.Text level="h5" color={10}>
      {message}
    </Text.Text>
    {description != null && (
      <Text.Text level="p" color={9}>
        {description}
      </Text.Text>
    )}
  </Flex.Box>
);
