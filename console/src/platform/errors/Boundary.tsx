// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, useCallback } from "react";

import { ErrorDiagnostics } from "@/platform/errors/ErrorDiagnostics";
import { Session } from "@/session";

export interface BoundaryProps extends PropsWithChildren {
  /** When provided, crash diagnostics include the layout page's name and key. */
  layoutKey?: string;
}

/** Error boundary that renders ErrorDiagnostics on a crash, annotating it with the
 * connected Core and, when layoutKey is given, the page's name and key. */
export const Boundary = ({ layoutKey, children }: BoundaryProps): ReactElement => {
  const name = Session.Layout.useSelectName(layoutKey ?? "");
  const renderFallback = useCallback(
    (props: Errors.FallbackProps) => (
      <ErrorDiagnostics
        page={layoutKey == null ? undefined : { name, key: layoutKey }}
        {...props}
      />
    ),
    [name, layoutKey],
  );
  return (
    <Errors.Boundary FallbackComponent={renderFallback}>{children}</Errors.Boundary>
  );
};
