// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import {
  ErrorBoundary,
  type ErrorBoundaryPropsWithComponent,
} from "react-error-boundary";

import { Fallback } from "@/errors/Fallback";

export interface BoundaryProps extends optional.Optional<
  ErrorBoundaryPropsWithComponent,
  "FallbackComponent"
> {}

/**
 * Error boundary component that catches errors in its children and displays a fallback
 * UI. Uses react-error-boundary internally.
 */
export const Boundary = ({
  FallbackComponent = Fallback,
  ...rest
}: BoundaryProps): ReactElement => (
  <ErrorBoundary {...rest} FallbackComponent={FallbackComponent} />
);
