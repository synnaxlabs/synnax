// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const operationsNav: PageNavNode = {
  key: "operations",
  name: "Test and Operations",
  children: [
    {
      name: "Conceptual Overview",
      href: "/roles/operations/overview",
      key: "/roles/operations/overview",
    },
    {
      name: "Console UI Overview",
      href: "/roles/operations/installation",
      key: "/roles/operations/installation",
    },
    {
      name: "Line Plot",
      href: "/roles/operations/line-plot",
      key: "/roles/operations/line-plot",
    },
    {
      name: "PID View",
      href: "/roles/operations/pid",
      key: "/roles/operations/pid",
    },
    {
      name: "Automated Control",
      href: "/roles/operations/automation",
      key: "/roles/operations/automation",
    },
  ],
};
