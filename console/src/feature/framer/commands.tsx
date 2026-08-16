// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { framer } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { Command } from "@/platform/command";
import { Framer } from "@/platform/framer";

const DeleteDataCommand = Command.create({
  key: "delete_data",
  name: "Delete data",
  icon: <Icon.Delete />,
  useOnSelect: Framer.useDeleteDataModal,
  useVisible: () => Access.useDeleteGranted(framer.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [DeleteDataCommand];
