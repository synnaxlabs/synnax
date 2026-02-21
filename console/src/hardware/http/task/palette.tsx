// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { SCAN_LAYOUT } from "@/hardware/http/task/Scan";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(task.TYPE_ONTOLOGY_ID);

export const CreateScanCommand = Palette.createSimpleCommand({
  key: "http-create-scan-task",
  name: "Create an HTTP Scan Task",
  icon: <Icon.Logo.HTTP />,
  layout: SCAN_LAYOUT,
  useVisible,
});

export const COMMANDS = [CreateScanCommand];
