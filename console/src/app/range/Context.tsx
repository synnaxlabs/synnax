// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { SNAPSHOT_SERVICES } from "@/app/range/snapshots";
import { Range } from "@/platform/range";

export interface ContextProps extends PropsWithChildren<{}> {}

export const Context = ({ children }: ContextProps): ReactElement => (
  <Range.SnapshotServicesProvider services={SNAPSHOT_SERVICES}>
    {children}
  </Range.SnapshotServicesProvider>
);
