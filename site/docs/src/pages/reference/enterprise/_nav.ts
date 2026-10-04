// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const ENTERPRISE_NAV: PageNavNode = {
  key: "enterprise",
  name: "Enterprise",
  edition: "enterprise",
  children: [
    {
      key: "/reference/enterprise",
      href: "/reference/enterprise",
      name: "Set up Enterprise",
    },
    {
      key: "/reference/enterprise/requirements",
      href: "/reference/enterprise/requirements",
      name: "Requirements",
    },
    {
      key: "/reference/enterprise/console",
      href: "/reference/enterprise/console",
      name: "Connect the Console",
    },
    {
      key: "/reference/enterprise/license",
      href: "/reference/enterprise/license",
      name: "Activate a license",
    },
    {
      key: "/reference/enterprise/upgrade",
      href: "/reference/enterprise/upgrade",
      name: "Upgrade",
    },
    {
      key: "/reference/enterprise/production",
      href: "/reference/enterprise/production",
      name: "Deploy to production",
    },
    {
      key: "/reference/enterprise/certificates",
      href: "/reference/enterprise/certificates",
      name: "Install certificates",
    },
    {
      key: "/reference/enterprise/clusters",
      href: "/reference/enterprise/clusters",
      name: "Clusters and nodes",
    },
    {
      key: "/reference/enterprise/users",
      href: "/reference/enterprise/users",
      name: "Users",
    },
    {
      key: "driver",
      name: "Standalone Driver",
      children: [
        {
          key: "/reference/enterprise/driver/install",
          href: "/reference/enterprise/driver/install",
          name: "Install",
        },
        {
          key: "/reference/enterprise/driver/configuration",
          href: "/reference/enterprise/driver/configuration",
          name: "Configuration",
        },
        {
          key: "/reference/enterprise/driver/cli-reference",
          href: "/reference/enterprise/driver/cli-reference",
          name: "CLI reference",
        },
      ],
    },
    {
      key: "/reference/enterprise/cli-reference",
      href: "/reference/enterprise/cli-reference",
      name: "CLI reference",
    },
    {
      key: "/reference/enterprise/systemd-service",
      href: "/reference/enterprise/systemd-service",
      name: "systemd service",
    },
  ],
};
