// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { LABELS_SECTION, useOpen } from "@/app/library/tab";
import { useLabelsVisible, useVisible } from "@/app/library/visible";
import { Command } from "@/platform/command";

const OpenCommand = Command.create({
  key: "open_library",
  name: "Open library",
  icon: <Icon.Library />,
  useVisible,
  useOnSelect: useOpen,
});

const useOpenLabels = (): (() => void) => useOpen(LABELS_SECTION);

const EditLabelsCommand = Command.create({
  key: "edit_labels",
  name: "Edit labels",
  icon: <Icon.Label />,
  useVisible: useLabelsVisible,
  useOnSelect: useOpenLabels,
});

export const COMMANDS = [OpenCommand, EditLabelsCommand];
