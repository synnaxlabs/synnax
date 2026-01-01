// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const SYS_ADMIN_NAV: PageNavNode = {
  key: "sys-admin",
  name: "System Administrators",
  children: [
    {
      name: "Deployment Considerations",
      href: "/guides/sys-admin/deployment-considerations",
      key: "/guides/sys-admin/deployment-considerations",
    },
    {
      name: "Deployment with Self-Signed Certificates",
      href: "/guides/sys-admin/deployment-self-signed",
      key: "/guides/sys-admin/deployment-self-signed",
    },
    {
      name: "Using Synnax with Multiple Users",
      href: "/guides/sys-admin/multiple-users",
      key: "/guides/sys-admin/multiple-users",
    },
  ],
};
