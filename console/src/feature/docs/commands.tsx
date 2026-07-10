// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Command } from "@/platform/command";
import { Docs } from "@/platform/docs";

export const ReadCommand = Command.create({
  key: "read_the_docs",
  name: "Read the documentation",
  icon: <Icon.QuestionMark />,
  useOnSelect: Command.createPlacerUseOnSelect(Docs.LAYOUT),
});

export const COMMANDS = [ReadCommand];
