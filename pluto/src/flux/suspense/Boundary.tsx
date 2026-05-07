// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status as xstatus } from "@synnaxlabs/x";
import {
  Component,
  type ErrorInfo,
  type ReactElement,
  type ReactNode,
  Suspense,
} from "react";

import { Summary } from "@/status/base/Summary";

const isStatus = (value: unknown): value is xstatus.Status =>
  typeof value === "object" &&
  value !== null &&
  "variant" in value &&
  "message" in value;

const stringifyThrown = (thrown: unknown): string => {
  if (thrown == null) return "Unknown error";
  if (typeof thrown === "string") return thrown;
  if (typeof thrown === "number" || typeof thrown === "boolean") return String(thrown);
  return "Unknown error";
};

const toStatus = (thrown: unknown): xstatus.Status => {
  if (isStatus(thrown)) return thrown;
  if (thrown instanceof Error) return xstatus.fromException(thrown, "Failed to load");
  return xstatus.fromException(new Error(stringifyThrown(thrown)), "Failed to load");
};

export type ErrorFallback = (status: xstatus.Status) => ReactNode;

interface ErrorBoundaryProps {
  fallback: ErrorFallback;
  children: ReactNode;
}

interface ErrorBoundaryState {
  status: xstatus.Status | null;
}

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { status: null };

  static getDerivedStateFromError(error: unknown): ErrorBoundaryState {
    return { status: toStatus(error) };
  }

  componentDidCatch(_error: unknown, _info: ErrorInfo): void {
    // Status is captured in getDerivedStateFromError; no-op here. Hook for
    // future telemetry (e.g., pushing to the Status.Aggregator) can live here.
  }

  render(): ReactNode {
    const { status } = this.state;
    if (status != null) return this.props.fallback(status);
    return this.props.children;
  }
}

export interface BoundaryProps {
  /// Rendered while children are suspended. Defaults to nothing (blank space).
  /// Pass a skeleton, spinner, or layout placeholder for better UX.
  loading?: ReactNode;
  /// Rendered when a child throws. Receives the thrown error or status,
  /// normalized to `xstatus.Status`. Defaults to `<Status.Summary>` rendering
  /// the status message and variant. Pass a custom render to override (e.g.,
  /// a "this resource was deleted" panel with a close-tab button).
  error?: ErrorFallback;
  children: ReactNode;
}

const defaultErrorFallback: ErrorFallback = (status) => <Summary status={status} />;

/// Suspense + ErrorBoundary in one consumer-facing component. Place per-panel
/// to scope independently-loadable regions of the UI: while the panel is
/// loading, `loading` shows; when a child throws, `error` shows.
///
/// Thrown values are normalized through the existing status pipeline. A typed
/// error with a `toStatus()` method is converted via `xstatus.fromException`.
/// A bare `Status` thrown directly is used as-is. The error boundary's
/// `error` prop receives an `xstatus.Status` either way, so the fallback
/// renders against one consistent shape.
export const Boundary = ({
  loading,
  error = defaultErrorFallback,
  children,
}: BoundaryProps): ReactElement => (
  <ErrorBoundary fallback={error}>
    <Suspense fallback={loading}>{children}</Suspense>
  </ErrorBoundary>
);
