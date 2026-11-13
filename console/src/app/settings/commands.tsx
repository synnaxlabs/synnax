// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { LABELS_SECTION, useOpen } from "@/app/settings/tab";
import { Command } from "@/platform/command";
import { Panel } from "@/platform/panel";

const OpenCommand = Command.create({
  key: "open_settings",
  name: "Open settings",
  icon: <Icon.Settings />,
  useVisible: Panel.useCanOpenTab,
  useOnSelect: useOpen,
});

const useLabelsVisible = (): boolean => {
  const canOpen = Panel.useCanOpenTab();
  return Access.useUpdateGranted(label.TYPE_RESOURCE_ID) && canOpen;
};

const useOpenLabels = (): (() => void) => useOpen(LABELS_SECTION);

const EditLabelsCommand = Command.create({
  key: "edit_labels",
  name: "Edit labels",
  icon: <Icon.Label />,
  useVisible: useLabelsVisible,
  useOnSelect: useOpenLabels,
});

export const COMMANDS = [OpenCommand, EditLabelsCommand];
