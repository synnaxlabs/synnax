// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors, Panel } from "@synnaxlabs/pluto";
import { type ComponentType, type ReactElement } from "react";

import { Disconnected, isConnectionError } from "@/platform/errors/Disconnected";
import { Session } from "@/session";

export interface ErrorDiagnosticsProps extends Errors.FallbackProps {
  /** When provided, the crashed panel's name and key are appended to the message. */
  panelKey?: string;
}

const panelLine = (panelKey?: string, name?: string): string | null => {
  if (panelKey == null) return null;
  return `${name != null ? `"${name}" ` : ""}(${panelKey})`;
};

// Annotates any caught error with the connected Core and, for a crashed panel, its name
// and key, so a crash screenshot is triageable. An unreachable Core is not a crash, so
// it gets the connection surface instead.
const ErrorDiagnostics = ({
  panelKey,
  error,
  ...rest
}: ErrorDiagnosticsProps): ReactElement => {
  const target = Session.Core.useSelectSelected();
  const core =
    target != null ? `${target.name} (${target.host}:${target.port})` : "none";
  const { data: crashed } = Panel.useResult(
    panelKey == null ? null : { key: panelKey },
  );
  if (isConnectionError(error)) return <Disconnected />;
  const message = [error.message, `Core: ${core}`, panelLine(panelKey, crashed?.name)]
    .filter((line): line is string => line != null)
    .join("\n");
  // Clone rather than mutate the caught error, preserving its name, stack, and cause.
  const displayError = new Error(message, { cause: error.cause });
  displayError.name = error.name;
  displayError.stack = error.stack;
  return <Errors.Fallback error={displayError} {...rest} />;
};

/** Builds the fallback component a boundary renders on a crash. */
export const useFallback =
  (panelKey?: string): ComponentType<Errors.FallbackProps> =>
  (props: Errors.FallbackProps) => <ErrorDiagnostics panelKey={panelKey} {...props} />;
