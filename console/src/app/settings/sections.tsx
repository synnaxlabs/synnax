// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, license } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { LABELS_SECTION } from "@/app/settings/tab";
import { Account } from "@/feature/account";
import { Core } from "@/feature/core";
import { Label } from "@/feature/label";
import { License } from "@/feature/license";
import { Persist } from "@/feature/persist";
import { Theme } from "@/feature/theme";
import { Version } from "@/feature/version";
import { Settings } from "@/platform/settings";

const Labels = (): ReactElement => (
  <Settings.Page title="Labels">
    <Label.Settings />
  </Settings.Page>
);

const useLabelsVisible = (): boolean => Access.useUpdateGranted(label.TYPE_RESOURCE_ID);

const Appearance = (): ReactElement => (
  <Settings.Page title="Appearance">
    <Theme.Settings />
  </Settings.Page>
);

const UsageData = (): ReactElement => (
  <Settings.Page title="Usage data">
    <Account.UsageDataSettings />
  </Settings.Page>
);

const Storage = (): ReactElement => (
  <Settings.Page title="Storage">
    <Persist.Settings />
  </Settings.Page>
);

const General = (): ReactElement => (
  <Settings.Page title="General">
    <Core.Settings />
    <Version.Settings />
  </Settings.Page>
);

const LicenseSection = (): ReactElement => (
  <Settings.Page title="License">
    <License.Info />
  </Settings.Page>
);

const useLicenseVisible = (): boolean => Access.useRetrieveGranted(license.RESOURCE_ID);

const PREFERENCES: Settings.Section[] = [
  {
    key: "appearance",
    name: "Appearance",
    icon: <Icon.DarkMode />,
    Content: Appearance,
  },
  ...(DESKTOP
    ? [
        {
          key: "usage_data",
          name: "Usage data",
          icon: <Icon.Analyze />,
          Content: UsageData,
        },
      ]
    : []),
  { key: "storage", name: "Storage", icon: <Icon.Database />, Content: Storage },
];

export const GROUPS: Settings.Group[] = [
  { key: "preferences", name: "Preferences", sections: PREFERENCES },
  {
    key: "core",
    name: "Core",
    sections: [
      { key: "general", name: "General", icon: <Icon.Core />, Content: General },
      {
        key: LABELS_SECTION,
        name: "Labels",
        icon: <Icon.Label />,
        Content: Labels,
        useVisible: useLabelsVisible,
      },
      {
        key: "license",
        name: "License",
        icon: <Icon.License />,
        Content: LicenseSection,
        useVisible: useLicenseVisible,
      },
    ],
  },
];
