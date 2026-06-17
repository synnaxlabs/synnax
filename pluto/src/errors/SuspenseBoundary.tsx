// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentType, type ReactElement, type ReactNode, Suspense } from "react";

import { Boundary } from "@/errors/Boundary";
import { type FallbackProps } from "@/errors/Fallback";

export interface SuspenseBoundaryProps {
  /// Rendered while children are suspended. Defaults to nothing (blank space).
  /// Pass a skeleton, spinner, or layout placeholder for better UX.
  loading?: ReactNode;
  /// Custom fallback component rendered when a child throws. Defaults to the diagnostic
  /// `<Errors.Fallback>` page. Flux retrieve errors throw an Error whose `cause` is the
  /// original `status.Status`, so a custom fallback can recover status fields via
  /// `(error.cause as Status)` when needed.
  FallbackComponent?: ComponentType<FallbackProps>;
  children: ReactNode;
}

/// `Errors.Boundary` + React `<Suspense>` in one consumer-facing component.
/// Place per-panel to scope independently-loadable regions of the UI: while
/// the panel is loading, `loading` shows; when a child throws, the fallback
/// shows.
export const SuspenseBoundary = ({
  loading,
  FallbackComponent,
  children,
}: SuspenseBoundaryProps): ReactElement => (
  <Boundary FallbackComponent={FallbackComponent}>
    <Suspense fallback={loading}>{children}</Suspense>
  </Boundary>
);
