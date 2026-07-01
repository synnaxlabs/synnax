// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Access, Schematic as PSchematic } from "@synnaxlabs/pluto";

import { Schematic as CSchematic } from "@/feature/schematic";
import { Palette } from "@/platform/palette";

const useCreate = () => CSchematic.useCreate({});

const CreateCommand = Palette.createCommand({
  key: "create_schematic",
  name: "Create a schematic",
  icon: <PSchematic.CreateIcon />,
  useOnSelect: useCreate,
  useVisible: () => Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand];
