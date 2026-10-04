// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const ALARMS_NAV: PageNavNode = {
  key: "alarms",
  name: "Alarms",
  children: [
    {
      key: "/reference/alarms",
      href: "/reference/alarms",
      name: "Overview",
    },
    {
      key: "/reference/alarms/statuses",
      href: "/reference/alarms/statuses",
      name: "Statuses",
    },
    {
      key: "/reference/alarms/redlines",
      href: "/reference/alarms/redlines",
      name: "Redlines",
    },
    {
      key: "/reference/alarms/arc",
      href: "/reference/alarms/arc",
      name: "Arc alarms",
    },
    {
      key: "/reference/alarms/pagerduty",
      href: "/reference/alarms/pagerduty",
      name: "PagerDuty",
    },
  ],
};
