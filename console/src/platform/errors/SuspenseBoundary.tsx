// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  Suspense,
} from "react";

import { Boundary } from "@/platform/errors/Boundary";

export interface SuspenseBoundaryProps extends PropsWithChildren {
  /** When provided, crash diagnostics include the layout page's name and key. */
  layoutKey?: string;
  /** Rendered while children are suspended. Defaults to blank space. */
  loading?: ReactNode;
}

/** Boundary + React Suspense in one component. On a crash it renders ErrorDiagnostics
 * annotated with the connected Core and, when layoutKey is given, the page details. */
export const SuspenseBoundary = ({
  layoutKey,
  loading,
  children,
}: SuspenseBoundaryProps): ReactElement => (
  <Boundary layoutKey={layoutKey}>
    <Suspense fallback={loading}>{children}</Suspense>
  </Boundary>
);
