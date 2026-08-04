// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useOpenWindow } from "@/feature/panel/useOpenWindow";
import { Command } from "@/platform/command";
import { Session } from "@/session";

const useOpenNewWindow = () => {
  const openWindow = useOpenWindow();
  const getSelected = Session.Panel.useGetSelected();
  return useCallback(() => openWindow(getSelected()), [openWindow, getSelected]);
};

const OpenWindowCommand = Command.create({
  key: "panel_open_window",
  name: "Open a new window",
  icon: <Icon.OpenInNewWindow />,
  useOnSelect: useOpenNewWindow,
});

export const COMMANDS = [OpenWindowCommand];
