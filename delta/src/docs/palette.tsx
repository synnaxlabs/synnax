// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { createDocsLayout } from "@/docs/layout";
import { Command } from "@/palette/Palette";

export const readTheDocsCommand: Command = {
  key: "read-the-docs",
  name: "Read the docs",
  icon: <Icon.QuestionMark />,
  onSelect: ({ placeLayout: layoutPlacer }) => layoutPlacer(createDocsLayout()),
};

export const DOCS_COMMANDS = [readTheDocsCommand];
