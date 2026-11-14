// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, label, license, secret, user } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { LABELS_SECTION } from "@/app/settings/tab";
import { Account } from "@/feature/account";
import { Core } from "@/feature/core";
import { Label } from "@/feature/label";
import { License } from "@/feature/license";
import { Persist } from "@/feature/persist";
import { Secret } from "@/feature/secret";
import { Theme } from "@/feature/theme";
import { User } from "@/feature/user";
import { Version } from "@/feature/version";
import { Settings } from "@/platform/settings";
import { Session } from "@/session";

const Profile = (): ReactElement => (
  <Settings.Page title="Profile">
    <User.ProfileSettings />
  </Settings.Page>
);

const APIKeys = (): ReactElement => (
  <Settings.Page title="API keys">
    <User.KeySettings />
  </Settings.Page>
);

const Users = (): ReactElement => (
  <Settings.Page title="Users">
    <User.Members />
  </Settings.Page>
);

const useUsersVisible = (): boolean => Access.useUpdateGranted(user.TYPE_RESOURCE_ID);

const Labels = (): ReactElement => (
  <Settings.Page title="Labels">
    <Label.Settings />
  </Settings.Page>
);

const useLabelsVisible = (): boolean => Access.useUpdateGranted(label.TYPE_RESOURCE_ID);

const CERTIFICATES_SECTION = "certificates";

const Secrets = (): ReactElement => {
  const dispatch = Session.useDispatch();
  return (
    <Settings.Page title="Secrets">
      <Secret.Secrets
        onSelectCertificate={(item) =>
          dispatch(
            Session.Nav.selectSettingsSection({ section: CERTIFICATES_SECTION, item }),
          )
        }
      />
    </Settings.Page>
  );
};

const useSecretsVisible = (): boolean =>
  Access.useRetrieveGranted(secret.TYPE_RESOURCE_ID);

const Certificates = (): ReactElement => {
  const dispatch = Session.useDispatch();
  return (
    <Settings.Page title="Certificates">
      <Secret.Certificates
        selected={Session.Nav.useSelectSettingsItem()}
        onShowSelected={() =>
          dispatch(Session.Nav.selectSettingsSection({ section: CERTIFICATES_SECTION }))
        }
      />
    </Settings.Page>
  );
};

const useCertificatesVisible = (): boolean =>
  Access.useRetrieveGranted(certificate.TYPE_RESOURCE_ID);

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

// Desktop runs a single-user embedded Core, so it has no accounts to manage.
const ACCOUNT: Settings.Group = {
  key: "account",
  name: "Account",
  sections: [
    { key: "profile", name: "Profile", icon: <Icon.User />, Content: Profile },
    { key: "api_keys", name: "API keys", icon: <Icon.Lock />, Content: APIKeys },
  ],
};

export const GROUPS: Settings.Group[] = [
  ...(DESKTOP ? [] : [ACCOUNT]),
  { key: "preferences", name: "Preferences", sections: PREFERENCES },
  {
    key: "core",
    name: "Core",
    sections: [
      { key: "general", name: "General", icon: <Icon.Core />, Content: General },
      ...(DESKTOP
        ? []
        : [
            {
              key: "users",
              name: "Users",
              icon: <Icon.User />,
              Content: Users,
              useVisible: useUsersVisible,
            },
          ]),
      {
        key: LABELS_SECTION,
        name: "Labels",
        icon: <Icon.Label />,
        Content: Labels,
        useVisible: useLabelsVisible,
      },
      {
        key: "secrets",
        name: "Secrets",
        icon: <Icon.Secret />,
        Content: Secrets,
        useVisible: useSecretsVisible,
      },
      {
        key: CERTIFICATES_SECTION,
        name: "Certificates",
        icon: <Icon.Certificate />,
        Content: Certificates,
        useVisible: useCertificatesVisible,
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
