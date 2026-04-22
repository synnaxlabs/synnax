// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const PAGERDUTY_NAV: PageNavNode = {
  key: "pagerduty",
  name: "PagerDuty",
  children: [
    {
      key: "/reference/driver/pagerduty/get-started",
      href: "/reference/driver/pagerduty/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/driver/pagerduty/alert-task",
      href: "/reference/driver/pagerduty/alert-task",
      name: "Alert Task",
    },
  ],
};
