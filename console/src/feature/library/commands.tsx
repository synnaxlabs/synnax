// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library } from "@synnaxlabs/client";
import { Access, Library } from "@synnaxlabs/pluto";

import { useCreate } from "@/feature/library/useCreate";
import { Command } from "@/platform/command";

export const CreateCommand = Command.create({
  key: "create_library",
  name: "Create library",
  icon: <Library.CreateIcon />,
  useVisible: () => Access.useCreateGranted(library.TYPE_ONTOLOGY_ID),
  useOnSelect: useCreate,
});

export const COMMANDS = [CreateCommand];
