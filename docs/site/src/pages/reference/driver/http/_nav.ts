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
      key: "/reference/driver/http/get-started",
      href: "/reference/driver/http/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/driver/http/connect-server",
      href: "/reference/driver/http/connect-server",
      name: "Connect to a Server",
    },
    {
      key: "/reference/driver/http/read-task",
      href: "/reference/driver/http/read-task",
      name: "Read Task",
    },
    {
      key: "/reference/driver/http/write-task",
      href: "/reference/driver/http/write-task",
      name: "Write Task",
    },
  ],
};
