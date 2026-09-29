// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const HTTP_NAV: PageNavNode = {
  key: "http",
  name: "HTTP",
  children: [
    {
      key: "/reference/hardware/http/connect-server",
      href: "/reference/hardware/http/connect-server",
      name: "Connect to a server",
    },
    {
      key: "/reference/hardware/http/read-task",
      href: "/reference/hardware/http/read-task",
      name: "Read task",
    },
    {
      key: "/reference/hardware/http/write-task",
      href: "/reference/hardware/http/write-task",
      name: "Write task",
    },
  ],
};
