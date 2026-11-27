// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { license } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { LICENSE_SECTION, useOpen } from "@/app/settings/tab";
import { Command } from "@/platform/command";

const OpenCommand = Command.create({
  key: "open_settings",
  name: "Open settings",
  icon: <Icon.Settings />,
  useOnSelect: useOpen,
});

const useLicenseVisible = (): boolean => Access.useRetrieveGranted(license.RESOURCE_ID);

const useOpenLicense = (): (() => void) => useOpen(LICENSE_SECTION);

const ShowLicenseCommand = Command.create({
  key: "open_license_info",
  name: "Show license info",
  icon: <Icon.License />,
  useVisible: useLicenseVisible,
  useOnSelect: useOpenLicense,
});

export const COMMANDS = [OpenCommand, ShowLicenseCommand];
