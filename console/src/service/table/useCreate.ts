// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Table } from "@synnaxlabs/pluto";

import { Project } from "@/project";
import { create } from "@/service/table/layout";

export const useCreate = Project.createUseCreate({
  useCreate: Table.useCreate,
  toCreateParams: ({ overrides, project }) => ({
    name: "Table",
    ...overrides,
    project,
  }),
  createSessionState: ({ key, name }) => create({ key, name, editable: true }),
});
