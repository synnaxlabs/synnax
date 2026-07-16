// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Session } from "@/session";

export interface PageContext {
  name?: string;
  key: string;
}

export interface ErrorDiagnosticsProps extends Errors.FallbackProps {
  page?: PageContext;
}

const pageLine = (page?: PageContext): string | null => {
  if (page == null) return null;
  const name = page.name != null ? `"${page.name}" ` : "";
  return `${name}(${page.key})`;
};

// Annotates any caught error with the connected Core and, for layout pages, the page's
// name and key, so a crash screenshot is triageable.
export const ErrorDiagnostics = ({
  page,
  error,
  ...rest
}: ErrorDiagnosticsProps): ReactElement => {
  const cluster = Session.Cluster.useSelectState();
  const core =
    cluster != null ? `${cluster.name} (${cluster.host}:${cluster.port})` : "none";
  const message = [error.message, `Core: ${core}`, pageLine(page)]
    .filter((line): line is string => line != null)
    .join("\n");
  // Clone rather than mutate the caught error, preserving its name, stack, and cause.
  const displayError = new Error(message, { cause: error.cause });
  displayError.name = error.name;
  displayError.stack = error.stack;
  return <Errors.Fallback error={displayError} {...rest} />;
};
