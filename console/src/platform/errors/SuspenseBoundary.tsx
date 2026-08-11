// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { useFallback } from "@/platform/errors/ErrorDiagnostics";

export interface SuspenseBoundaryProps extends PropsWithChildren {
  /** When provided, crash diagnostics include the crashed panel's name and key. */
  panelKey?: string;
  /** Rendered while children are suspended. Defaults to a delayed loading indicator. */
  loading?: ReactNode;
}

/** Boundary + React Suspense in one component. On a crash it renders ErrorDiagnostics
 * annotated with the connected Core and, when panelKey is given, the panel details. */
export const SuspenseBoundary = ({
  panelKey,
  loading,
  children,
}: SuspenseBoundaryProps): ReactElement => (
  <Errors.SuspenseBoundary FallbackComponent={useFallback(panelKey)} loading={loading}>
    {children}
  </Errors.SuspenseBoundary>
);
