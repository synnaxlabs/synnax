// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { video } from "@synnaxlabs/client";
import { Access, Video } from "@synnaxlabs/pluto";

import { useCreate } from "@/feature/video/useCreate";
import { FLAGS } from "@/flags";
import { Command } from "@/platform/command";

const CreateCommand = Command.create({
  key: "create_video",
  name: "Create video view",
  icon: <Video.CreateIcon />,
  useOnSelect: useCreate,
  useVisible: () => Access.useCreateGranted(video.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = FLAGS.video ? [CreateCommand] : [];
