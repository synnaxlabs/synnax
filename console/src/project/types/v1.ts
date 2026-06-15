// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";

import type * as v0 from "@/project/types/v0";

export interface Project extends Omit<project.Project, "layout"> {}

export const VERSION = "1.0.0";
type Version = typeof VERSION;

export interface SliceState extends migrate.Migratable<Version> {
  active: Project | null;
}

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  active: null,
};

export const SLICE_MIGRATION_NAME = "project.slice";

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: SLICE_MIGRATION_NAME,
  migrate: ({ active, projects }) => {
    if (active == null) return { active: null, version: VERSION };
    const found = projects[active];
    return {
      active: found != null ? { name: found.name, key: active } : null,
      version: VERSION,
    };
  },
});
