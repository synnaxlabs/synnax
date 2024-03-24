// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const sysAdminNav: PageNavNode = {
  key: "sys-admin",
  name: "System Administrators",
  children: [
    {
      name: "Concepts",
      href: "/roles/sys-admin/concepts",
      key: "/roles/sys-admin/concepts",
    },
    {
      name: "Deployment with Self-Signed Certificates",
      href: "/roles/sys-admin/deployment-self-signed",
      key: "/roles/sys-admin/deployment-self-signed",
    },
  ],
};
