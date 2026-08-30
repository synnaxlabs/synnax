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
import { errors, primitive, type record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { z } from "zod";

import { Breadcrumb } from "@/breadcrumb";
import { Button } from "@/button";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { type ResolvedStack, resolveStack } from "@/errors/resolveStack";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
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
  /** Extra information to copying to the clipboard when the user clicks the "Copy"
   * button. */
  extraInfo?: record.Unknown;
}

/** Longest chain of causes a report walks, in case one links back to itself. */
const MAX_CAUSE_DEPTH = 10;

/**
 * @returns one line per `cause` beneath err, outermost first. A wrapper carries only
 * its own message, so the reason the report is about is the last line.
 */
const causeChain = (err: Error): string[] => {
  const lines: string[] = [];
  let cause = err.cause;
  while (cause != null && lines.length < MAX_CAUSE_DEPTH) {
    const asError = errors.fromUnknown(cause);
    lines.push(
      asError instanceof z.core.$ZodError
        ? z.prettifyError(asError)
        : `${asError.name}: ${asError.message}`,
    );
    cause = cause instanceof Error ? cause.cause : null;
  }
  return lines;
};

/**
 * Default error fallback component. Can be used standalone or with ErrorBoundary.
 * Supports both compact (for mosaic leafs) and full (for page overlays) variants.
 *
 * @example
 * // With default retry button
 * <Fallback error={error} resetErrorBoundary={reset} />
 * @example
 * // With custom actions
 * <Fallback error={error} resetErrorBoundary={reset} icon={<Logo />}>
 *   <Button onClick={reset}>Try again</Button>
 *   <Button onClick={clear}>Clear storage</Button>
 * </Fallback>
 */
export const Fallback = ({
  error,
  componentStack,
  resetErrorBoundary,
  children = <DefaultChild resetErrorBoundary={resetErrorBoundary} />,
  extraInfo,
}: FallbackProps): ReactElement => {
  const [resolved, setResolved] = useState<ResolvedStack | null>(null);
  useEffect(() => {
    let cancelled = false;
    resolveStack(error, componentStack ?? null)
      .then((r) => {
        if (!cancelled) setResolved(r);
      })
      .catch((cause: unknown) => {
        console.warn("Unexpected source-map resolution failure", cause);
      });
    return () => {
      cancelled = true;
    };
  }, [error, componentStack]);

  const displayStack = resolved?.stack || error.stack || null;
  const displayComponentStack = resolved?.componentStack || componentStack || null;

  // A raw ZodError's message is its entire issues array as JSON; prettify it for
  // display. Copy diagnostics keeps the raw message for full fidelity.
  const message =
    error instanceof z.core.$ZodError ? z.prettifyError(error) : error.message;
  const multiline = message.includes("\n");
  const causes = useMemo(() => causeChain(error), [error]);

  const getCopyText = useCallback(() => {
    const sections: string[] = [];
    sections.push(`Error: ${error.name}`);
    sections.push(`Message: ${error.message}`);
    if (causes.length > 0) sections.push(`\nCaused by:\n${causes.join("\n")}`);
    if (displayStack) sections.push(`\nStack trace:\n${displayStack}\n`);
    if (displayComponentStack)
      sections.push(`\nComponent stack:\n${displayComponentStack}`);
    if (extraInfo && Object.keys(extraInfo).length > 0)
      sections.push(`\nAdditional info:\n${JSON.stringify(extraInfo, null, 2)}`);
    return sections.join("\n");
  }, [error, causes, displayStack, displayComponentStack, extraInfo]);

  return (
    <Flex.Box className={CSS.BE("error-fallback", "container")} y grow center>
      <Flex.Box
        background={2}
        className={CSS.BE("error-fallback", "content")}
        bordered
        borderColor="var(--pluto-error-z-40)"
        borderWidth={1}
        rounded="large"
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
              level={multiline ? "small" : "h5"}
              variant={multiline ? "code" : undefined}
              color={10}
              className={CSS.cls(
                CSS.BE("error-fallback", "message"),
                multiline && CSS.BEM("error-fallback", "message", "multiline"),
              )}
            >
              {message}
            </Text.Text>
            {causes.length > 0 && (
              <Text.Text
                level="small"
                variant="code"
                color={9}
                className={CSS.BE("error-fallback", "causes")}
              >
                {causes.map((cause) => `Caused by: ${cause}`).join("\n")}
              </Text.Text>
            )}
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
            {primitive.isNonZero(displayComponentStack || displayStack) && (
              <Text.Text
                className={CSS.BE("error-fallback", "stack")}
                level="small"
                color={9}
              >
                {displayComponentStack || displayStack}
              </Text.Text>
            )}
          </Flex.Box>
          <Divider.Divider x />
          <Flex.Box justify="between" x>
            <Button.Copy text={getCopyText} textColor={10}>
              Copy diagnostics
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
  <Button.Button variant="filled" onClick={resetErrorBoundary}>
    Reload
  </Button.Button>
);
