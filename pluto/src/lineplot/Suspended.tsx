// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { useEnsureRetrieved } from "@/lineplot/queries";
import { Scope } from "@/lineplot/scope";

export const useKey = Scope.use;

export interface SuspendedProps extends PropsWithChildren {
  linePlotKey: lineplot.Key;
}

export const Suspended = ({ linePlotKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: linePlotKey });
  return <Scope.Provider value={linePlotKey}>{children}</Scope.Provider>;
};
