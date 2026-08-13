// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, type Synnax as Client } from "@synnaxlabs/client";
import { Errors, Synnax } from "@synnaxlabs/pluto";
import { type ComponentType, type ReactElement, useCallback } from "react";

import { Session } from "@/session";

export interface ErrorDiagnosticsProps extends Errors.FallbackProps {
  /** When provided, the crashed panel's name and key are appended to the message. */
  panelKey?: string;
}

// panelLine resolves panelKey's name for display. The panel backing a crashed tab may
// not be cached, so a miss falls back to the key alone rather than throwing out of the
// crash fallback itself.
const panelLine = (client: Client | null, panelKey?: string): string | null => {
  if (panelKey == null) return null;
  const cached = client?.panels.getCached(panelKey);
  const name = query.isLive(cached) ? cached.name : undefined;
  return `${name != null ? `"${name}" ` : ""}(${panelKey})`;
};

// Annotates any caught error with the connected Core and, for a crashed panel, its name
// and key, so a crash screenshot is triageable.
export const ErrorDiagnostics = ({
  panelKey,
  error,
  ...rest
}: ErrorDiagnosticsProps): ReactElement => {
  const cluster = Session.Cluster.useSelectState();
  const core =
    cluster != null ? `${cluster.name} (${cluster.host}:${cluster.port})` : "none";
  const client = Synnax.use();
  const message = [error.message, `Core: ${core}`, panelLine(client, panelKey)]
    .filter((line): line is string => line != null)
    .join("\n");
  // Clone rather than mutate the caught error, preserving its name, stack, and cause.
  const displayError = new Error(message, { cause: error.cause });
  displayError.name = error.name;
  displayError.stack = error.stack;
  return <Errors.Fallback error={displayError} {...rest} />;
};

/** Builds the fallback component a boundary renders on a crash. */
export const useFallback = (panelKey?: string): ComponentType<Errors.FallbackProps> =>
  useCallback(
    (props: Errors.FallbackProps) => (
      <ErrorDiagnostics panelKey={panelKey} {...props} />
    ),
    [panelKey],
  );
