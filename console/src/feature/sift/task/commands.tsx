// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { useCreateStream } from "@/feature/sift/task/Stream";
import { Command } from "@/platform/command";

const useVisible = () => Access.useCreateGranted(task.TYPE_RESOURCE_ID);

const CreateStreamCommand = Command.create({
  key: "sift_create_stream_task",
  name: "Create Sift stream task",
  icon: <Icon.Logo.Sift />,
  useOnSelect: useCreateStream,
  useVisible,
});

export const COMMANDS = [CreateStreamCommand];
