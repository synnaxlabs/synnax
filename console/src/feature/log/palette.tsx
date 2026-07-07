// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access, Log } from "@synnaxlabs/pluto";

import { Log as PlatformLog } from "@/platform/log";
import { Palette } from "@/platform/palette";

const CreateCommand = Palette.createCommand({
  key: "create_log",
  name: "Create a log",
  icon: <Log.CreateIcon />,
  useOnSelect: PlatformLog.useCreate,
  useVisible: () => Access.useCreateGranted(log.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand];
