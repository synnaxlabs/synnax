// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { useEnsureRetrieved } from "@/arc/queries";
import { Scope } from "@/arc/scope";

export const useKey = Scope.use;

export interface SuspendedProps extends PropsWithChildren {
  arcKey: arc.Key;
}

export const Suspended = ({ arcKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: arcKey });
  return <Scope.Provider value={arcKey}>{children}</Scope.Provider>;
};
