// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Log } from "@synnaxlabs/pluto";

import { create } from "@/component/log/layout";
import { Project } from "@/project";

export const useCreate = Project.createUseCreate({
  useCreate: Log.useCreate,
  toCreateParams: ({ overrides, project }) => ({
    name: "Log",
    ...overrides,
    project,
  }),
  createSessionState: create,
});
