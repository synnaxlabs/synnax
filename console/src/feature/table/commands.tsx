// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { Access, Table } from "@synnaxlabs/pluto";

import { Command } from "@/platform/command";
import { Table as PlatformTable } from "@/platform/table";

const useCreate = () => PlatformTable.useCreate({});

const CreateCommand = Command.create({
  key: "create_table",
  name: "Create a table",
  icon: <Table.CreateIcon />,
  useOnSelect: useCreate,
  useVisible: () => Access.useCreateGranted(table.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand];
