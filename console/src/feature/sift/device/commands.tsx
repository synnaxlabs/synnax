// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { useConnectModal } from "@/feature/sift/device/useConnectModal";
import { FLAGS } from "@/flags";
import { Command } from "@/platform/command";

const useVisible = () => Access.useCreateGranted(device.TYPE_RESOURCE_ID);

const ConnectCommand = Command.create({
  key: "sift_connect",
  name: "Connect Sift",
  icon: <Icon.Logo.Sift />,
  useOnSelect: useConnectModal,
  useVisible,
});

export const COMMANDS = FLAGS.sift ? [ConnectCommand] : [];
