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
import { primitive } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Text } from "@/text";

/** Props for the error fallback component. */
export interface FallbackProps extends PropsWithChildren {
  /** The error that was caught. */
  error: Error;
  /** Function to reset the error boundary and retry rendering. */
  resetErrorBoundary: () => void;
  /** Variant of the fallback. */
  variant?: "compact" | "full";
  /** Whether to show the Synnax logo above the error details. Defaults to false. */
  showLogo?: boolean;
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
  resetErrorBoundary,
  variant = "compact",
  showLogo = false,
  children = <DefaultChild resetErrorBoundary={resetErrorBoundary} />,
}: FallbackProps): ReactElement => {
  const isCompact = variant === "compact";
  return (
    <Flex.Box
      className={CSS.BE("error-fallback")}
      y
      grow
      gap={isCompact ? "small" : "medium"}
      center
    >
      {showLogo && <Logo variant="icon" />}
      <Text.Text level={isCompact ? "h3" : "h1"} status="error">
        {error.name}
      </Text.Text>
      <Text.Text level={isCompact ? "p" : "h3"} color="var(--pluto-gray-l8)">
        {error.message}
      </Text.Text>
      {primitive.isNonZero(error.stack) && (
        <Text.Text
          className={CSS.BE("error-fallback", "stack")}
          level="small"
          style={{ whiteSpace: "pre-wrap" }}
          color="var(--pluto-gray-l6)"
        >
          {error.stack}
        </Text.Text>
      )}
      {children}
    </Flex.Box>
  );
};

const DefaultChild = ({
  resetErrorBoundary,
}: Pick<FallbackProps, "resetErrorBoundary">): ReactElement => (
  <Button.Button variant="outlined" size="small" onClick={resetErrorBoundary}>
    Reload
  </Button.Button>
);
