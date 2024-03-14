// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const analystsNav: PageNavNode = {
  key: "Sys Admins",
  name: "Sys Admins",
  children: [
    {
      name: "Quick Start",
      href: "/roles/sys-admin/quick-start",
      key: "/roles/sys-admin/quick-start",
    },
    {
      name: "Concepts",
      href: "/roles/sys-admin/concepts",
      key: "/roles/sys-admin/concepts",
    },
    {
      name: "Production Deployment",
      href: "/roles/sys-admin/production-deployment",
      key: "/roles/sys-admin/production-deployment",
    },
    {
      name: "Hardware Requirements",
      href: "/roles/sys-admin/hardware-requirements",
      key: "/roles/sys-admin/hardware-requirements",
    },
    {
      name: "CLI Reference",
      href: "/roles/sys-admin/cli-reference",
      key: "/roles/sys-admin/cli-reference",
    },
  ],
};
