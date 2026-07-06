// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Docs } from "@/platform/docs";
import { Palette } from "@/platform/palette";

export const ReadCommand = Palette.createSimpleCommand({
  key: "read_the_docs",
  name: "Read the documentation",
  icon: <Icon.QuestionMark />,
  layout: Docs.LAYOUT,
});

export const COMMANDS = [ReadCommand];
