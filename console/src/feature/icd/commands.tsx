// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, ICD } from "@synnaxlabs/pluto";

import { useCreate } from "@/feature/icd/useCreate";
import { useImportNew } from "@/feature/icd/useImport";
import { Command } from "@/platform/command";

const CreateCommand = Command.create({
  key: "create_icd",
  name: "Create ICD",
  icon: <ICD.CreateIcon />,
  useVisible: () => Access.useCreateGranted(icd.TYPE_RESOURCE_ID),
  useOnSelect: useCreate,
});

const ImportCommand = Command.create({
  key: "import_icd",
  name: "Import ICD",
  icon: <Icon.Import />,
  useVisible: () => Access.useCreateGranted(icd.TYPE_RESOURCE_ID),
  useOnSelect: useImportNew,
});

export const COMMANDS = [CreateCommand, ImportCommand];
