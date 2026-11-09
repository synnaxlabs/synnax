// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const ENTERPRISE_NAV: Nav.PageNavNode = {
  key: "enterprise",
  name: "Enterprise",
  edition: "enterprise",
  children: [
    {
      href: "/reference/enterprise",
      name: "Set up Enterprise",
    },
    {
      href: "/reference/enterprise/requirements",
      name: "Requirements",
    },
    {
      href: "/reference/enterprise/console",
      name: "Connect the Console",
    },
    {
      href: "/reference/enterprise/license",
      name: "Activate a license",
    },
    {
      href: "/reference/enterprise/upgrade",
      name: "Upgrade",
    },
    {
      key: "tls",
      name: "TLS",
      children: [
        {
          href: "/reference/enterprise/tls",
          name: "Set up TLS",
        },
        {
          href: "/reference/enterprise/certificates",
          name: "Install certificates",
        },
        {
          href: "/reference/enterprise/listeners",
          name: "Serve multiple listeners",
        },
      ],
    },
    {
      href: "/reference/enterprise/users",
      name: "Users",
    },
    {
      href: "/reference/enterprise/service",
      name: "Run as a service",
    },
    {
      href: "/reference/enterprise/production-checklist",
      name: "Production checklist",
    },
    {
      key: "driver",
      name: "Standalone Driver",
      children: [
        {
          href: "/reference/enterprise/driver/install",
          name: "Install",
        },
        {
          href: "/reference/enterprise/driver/configuration",
          name: "Configuration",
        },
        {
          href: "/reference/enterprise/driver/cli-reference",
          name: "CLI reference",
        },
      ],
    },
    {
      href: "/reference/enterprise/cli-reference",
      name: "CLI reference",
    },
    {
      href: "/reference/enterprise/troubleshooting",
      name: "Troubleshooting",
    },
  ],
};
