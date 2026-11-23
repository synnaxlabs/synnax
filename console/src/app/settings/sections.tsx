// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  certificate,
  color,
  enumeration,
  label,
  license,
  secret,
  user,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { LABELS_SECTION, LICENSE_SECTION } from "@/app/settings/tab";
import { Account } from "@/feature/account";
import { Color } from "@/feature/color";
import { Core } from "@/feature/core";
import { Enumeration } from "@/feature/enumeration";
import { Label } from "@/feature/label";
import { License } from "@/feature/license";
import { Persist } from "@/feature/persist";
import { Secret } from "@/feature/secret";
import { Theme } from "@/feature/theme";
import { User } from "@/feature/user";
import { Version } from "@/feature/version";
import type { Settings } from "@/platform/settings";
import { Session } from "@/session";

const useUsersVisible = (): boolean => Access.useUpdateGranted(user.TYPE_RESOURCE_ID);

const useLabelsVisible = (): boolean => Access.useUpdateGranted(label.TYPE_RESOURCE_ID);

const useColorsVisible = (): boolean => Access.useUpdateGranted(color.TYPE_RESOURCE_ID);

const useEnumerationsVisible = (): boolean =>
  Access.useUpdateGranted(enumeration.TYPE_RESOURCE_ID);

const CERTIFICATES_SECTION = "certificates";

const Secrets = (): ReactElement => {
  const dispatch = Session.useDispatch();
  return (
    <Secret.Secrets
      onSelectCertificate={(item) =>
        dispatch(
          Session.Nav.selectSettingsSection({ section: CERTIFICATES_SECTION, item }),
        )
      }
    />
  );
};

const useSecretsVisible = (): boolean =>
  Access.useRetrieveGranted(secret.TYPE_RESOURCE_ID);

const Certificates = (): ReactElement => {
  const dispatch = Session.useDispatch();
  return (
    <Secret.Certificates
      selected={Session.Nav.useSelectSettingsItem()}
      onShowSelected={() =>
        dispatch(Session.Nav.selectSettingsSection({ section: CERTIFICATES_SECTION }))
      }
    />
  );
};

const useCertificatesVisible = (): boolean =>
  Access.useRetrieveGranted(certificate.TYPE_RESOURCE_ID);

const About = (): ReactElement => (
  <>
    <Core.Settings />
    <Version.Settings />
  </>
);

const useLicenseVisible = (): boolean => Access.useRetrieveGranted(license.RESOURCE_ID);

const CONSOLE: Settings.Section[] = [
  {
    key: "appearance",
    name: "Appearance",
    icon: <Icon.DarkMode />,
    Content: Theme.Settings,
  },
  {
    key: "storage",
    name: "Storage",
    icon: <Icon.Database />,
    Content: Persist.Settings,
  },
];

// An embedded Core has one user, so its account is the Synnax account that the machine
// signs in with.
const ACCOUNT: Settings.Group = {
  key: "account",
  name: "Account",
  sections: Session.Runtime.CORE_EMBEDDED
    ? [
        {
          key: "usage_data",
          name: "Usage data",
          icon: <Icon.UsageData />,
          Content: Account.UsageDataSettings,
        },
      ]
    : [
        {
          key: "profile",
          name: "Profile",
          icon: <Icon.User />,
          Content: User.ProfileSettings,
        },
        {
          key: "api_keys",
          name: "API keys",
          icon: <Icon.APIKey />,
          Content: User.KeySettings,
        },
      ],
};

export const GROUPS: Settings.Group[] = [
  ACCOUNT,
  { key: "console", name: "Console", sections: CONSOLE },
  {
    key: "core",
    name: "Core",
    sections: [
      { key: "about", name: "About", icon: <Icon.About />, Content: About },
      ...(Session.Runtime.CORE_EMBEDDED
        ? []
        : [
            {
              key: "users",
              name: "Users",
              icon: <Icon.User />,
              Content: User.Members,
              useVisible: useUsersVisible,
            },
          ]),
      {
        key: LABELS_SECTION,
        name: "Labels",
        icon: <Icon.Label />,
        Content: Label.Settings,
        useVisible: useLabelsVisible,
      },
      {
        key: "colors",
        name: "Colors",
        icon: <Icon.Swatches />,
        Content: Color.Settings,
        useVisible: useColorsVisible,
      },
      {
        key: "enumerations",
        name: "Enumerations",
        icon: <Icon.Enumeration />,
        Content: Enumeration.Settings,
        useVisible: useEnumerationsVisible,
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
        key: LICENSE_SECTION,
        name: "License",
        icon: <Icon.License />,
        Content: License.Info,
        useVisible: useLicenseVisible,
      },
    ],
  },
];
