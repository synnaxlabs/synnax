// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const DESKTOP_NAV: Nav.PageNavNode = {
  key: "desktop",
  name: "Desktop",
  edition: "desktop",
  children: [
    {
      href: "/reference/desktop/troubleshooting",
      name: "Troubleshooting",
    },
    {
      href: "/reference/desktop/data",
      name: "Data",
    },
    {
      href: "/reference/desktop/uninstall",
      name: "Uninstall",
    },
  ],
};
