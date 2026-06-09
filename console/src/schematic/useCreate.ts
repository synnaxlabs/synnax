// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Schematic } from "@synnaxlabs/pluto";

import { Project } from "@/project";
import { create } from "@/schematic/layout";

export const useCreate = Project.createUseCreate({
  useCreate: Schematic.useCreate,
  createSessionState: ({ key, name }) => create({ key, name, editable: true }),
  defaultName: "Schematic",
  ontologyID: schematic.ontologyID,
});
