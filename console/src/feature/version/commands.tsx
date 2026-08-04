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
import { Version } from "@/platform/version";
import { Session } from "@/session";

const useVisible = () => Session.Runtime.ENGINE === "tauri";

export const OpenInfoCommand = Command.create({
  key: "open_version_info",
  name: "Show version info",
  icon: <Icon.Info />,
  useVisible,
  useOnSelect: Version.useInfoModal,
});

export const UpdateCommand = Command.create({
  key: "update_console",
  name: "Update & restart the Console",
  icon: <Icon.Download />,
  useVisible,
  useOnSelect: () => {
    const openInfo = Version.useInfoModal();
    return () => openInfo({ autoUpdate: true });
  },
});

export const COMMANDS = [OpenInfoCommand, UpdateCommand];
