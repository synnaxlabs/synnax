// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/errors/Fallback.css";

import { Logo } from "@synnaxlabs/media";
import { primitive, type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useCallback } from "react";

import { Breadcrumb } from "@/breadcrumb";
import { Button } from "@/button";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
// NOTE: Import Bar directly to avoid circular dependency (Nav.Drawer -> Errors -> Fallback)
import { Bar } from "@/nav/Bar";
import { Text } from "@/text";

/** Props for the error fallback component. */
export interface FallbackProps extends PropsWithChildren {
  /** The error that was caught. */
  error: Error;
  /** The React component stack trace from the error boundary. */
  componentStack?: string | null;
  /** Function to reset the error boundary and retry rendering. */
  resetErrorBoundary: () => void;
  /** Whether to show the Synnax logo above the error details. Defaults to false. */
  showLogo?: boolean;
  /** Extra information to copying to the clipboard when the user clicks the "Copy" button. */
  extraInfo?: record.Unknown;
}

/**
 * Default error fallback component. Can be used standalone or with ErrorBoundary.
 * Supports both compact (for mosaic leafs) and full (for page overlays) variants.
 *
 * @example
 * // With default retry button
 * <Fallback error={error} resetErrorBoundary={reset} />
 *
 * @example
 * // With custom actions
 * <Fallback error={error} resetErrorBoundary={reset} icon={<Logo />}>
 *   <Button onClick={reset}>Try Again</Button>
 *   <Button onClick={clear}>Clear Storage</Button>
 * </Fallback>
 */
export const Fallback = ({
  error,
  componentStack,
  resetErrorBoundary,
  children = <DefaultChild resetErrorBoundary={resetErrorBoundary} />,
  extraInfo,
}: FallbackProps): ReactElement => {
  const getCopyText = useCallback(() => {
    const sections: string[] = [];
    sections.push(`Error: ${error.name}`);
    sections.push(`Message: ${error.message}`);
    if (error.stack) sections.push(`\nStack Trace:\n${error.stack}\n`);
    if (componentStack) sections.push(`\nComponent Stack:\n${componentStack}`);
    if (extraInfo && Object.keys(extraInfo).length > 0)
      sections.push(`\nAdditional Info:\n${JSON.stringify(extraInfo, null, 2)}`);
    return sections.join("\n");
  }, [error, componentStack, extraInfo]);

  return (
    <Flex.Box className={CSS.BE("error-fallback", "container")} y grow center>
      <Flex.Box
        background={2}
        rounded
        className={CSS.BE("error-fallback", "content")}
        bordered
        borderColor={5}
        empty
      >
        <Bar location="top" bordered size="6rem">
          <Bar.Start className={CSS.BE("error-fallback", "nav-start")}>
            <Breadcrumb.Breadcrumb gap="tiny">
              <Breadcrumb.Segment color={9}>
                <Icon.Err />
              </Breadcrumb.Segment>
              <Breadcrumb.Segment
                color={9}
                className={CSS.BE("error-fallback", "header-text")}
              >
                Something went wrong
              </Breadcrumb.Segment>
            </Breadcrumb.Breadcrumb>
          </Bar.Start>
          <Bar.End className={CSS.BE("error-fallback", "nav-end")}>
            <Logo variant="icon" />
          </Bar.End>
        </Bar>
        <Flex.Box className={CSS.BE("error-fallback", "body")}>
          <Flex.Box>
            <Text.Text
              level="h3"
              status="error"
              className={CSS.BE("error-fallback", "name")}
            >
              {error.name}
            </Text.Text>
            <Text.Text
              level="h5"
              color={10}
              className={CSS.BE("error-fallback", "message")}
            >
              {error.message}
            </Text.Text>
          </Flex.Box>
          <Divider.Divider x />
          <Text.Text level="h5" color={9}>
            Stack trace
          </Text.Text>
          <Flex.Box
            rounded
            className={CSS.BE("error-fallback", "stack-container")}
            background={1}
            bordered
          >
            {primitive.isNonZero(componentStack || error.stack) && (
              <Text.Text
                className={CSS.BE("error-fallback", "stack")}
                level="small"
                color={9}
              >
                {componentStack || error.stack}
              </Text.Text>
            )}
          </Flex.Box>
          <Divider.Divider x />
          <Flex.Box justify="between" x>
            <Button.Copy text={getCopyText} textColor={10}>
              Copy Diagnostics
            </Button.Copy>
            <Flex.Box x>{children}</Flex.Box>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

const DefaultChild = ({
  resetErrorBoundary,
}: Pick<FallbackProps, "resetErrorBoundary">): ReactElement => (
  <Button.Button variant="filled" size="small" onClick={resetErrorBoundary}>
    Reload
  </Button.Button>
);
