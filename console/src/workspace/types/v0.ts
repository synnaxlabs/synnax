// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type workspace } from "@synnaxlabs/client";
import { type migrate } from "@synnaxlabs/x";

export const VERSION = "0.0.0";
type Version = typeof VERSION;

export interface SliceState extends migrate.Migratable<Version> {
  active: string | null;
  workspaces: Record<string, workspace.Workspace>;
}

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  active: null,
  workspaces: {},
};
