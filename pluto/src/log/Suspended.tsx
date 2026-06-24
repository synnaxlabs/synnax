// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type log } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { useEnsureRetrieved } from "@/log/queries";
import { Scope } from "@/log/scope";

export const useKey = Scope.use;

export interface SuspendedProps extends PropsWithChildren {
  logKey: log.Key;
}

export const Suspended = ({ logKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: logKey });
  return <Scope.Provider value={logKey}>{children}</Scope.Provider>;
};
