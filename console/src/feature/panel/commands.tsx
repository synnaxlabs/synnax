// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { openIconPicker } from "@/feature/panel/IconPicker";
import { OPEN_WINDOW_TRIGGER, useOpenWindow } from "@/feature/panel/useOpenWindow";
import { Command } from "@/platform/command";
import { Session } from "@/session";

const useOpenNewWindow = () => {
  const openWindow = useOpenWindow();
  const getSelected = Session.Panel.useGetSelected();
  return () => openWindow(getSelected());
};

const OpenWindowCommand = Command.create({
  key: "panel_open_window",
  name: "Open new window",
  icon: <Icon.OpenInNewWindow />,
  useOnSelect: useOpenNewWindow,
  useVisible: () => Session.Runtime.ENGINE === "tauri",
  trigger: OPEN_WINDOW_TRIGGER,
});

const useOpenIconPicker = () => {
  const getSelected = Session.Panel.useGetSelected();
  return () => {
    const selected = getSelected();
    if (selected != null) openIconPicker(selected);
  };
};

const useCanChangeIcon = (): boolean => {
  const selected = Session.Panel.useSelectSelected();
  const canEdit = Access.useUpdateGranted(
    panel.ontologyID(selected == null ? [] : [selected]),
  );
  return selected != null && canEdit;
};

const ChangeIconCommand = Command.create({
  key: "panel_change_icon",
  name: "Change panel icon",
  icon: <Icon.FillColor />,
  useOnSelect: useOpenIconPicker,
  useVisible: useCanChangeIcon,
});

export const COMMANDS = [OpenWindowCommand, ChangeIconCommand];
