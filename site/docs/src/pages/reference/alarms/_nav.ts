// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const ALARMS_NAV: Nav.PageNavNode = {
  key: "alarms",
  name: "Alarms",
  children: [
    {
      href: "/reference/alarms",
      name: "Overview",
    },
    {
      href: "/reference/alarms/statuses",
      name: "Statuses",
    },
    {
      href: "/reference/alarms/redlines",
      name: "Redlines",
    },
    {
      href: "/reference/alarms/arc",
      name: "Arc alarms",
    },
    {
      href: "/reference/alarms/pagerduty",
      name: "PagerDuty",
    },
  ],
};
