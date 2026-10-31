// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const HTTP_NAV: Nav.PageNavNode = {
  key: "http",
  name: "HTTP",
  children: [
    {
      href: "/reference/hardware/http/connect-server",
      name: "Connect to a server",
    },
    {
      href: "/reference/hardware/http/read-task",
      name: "Read task",
    },
    {
      href: "/reference/hardware/http/write-task",
      name: "Write task",
    },
  ],
};
