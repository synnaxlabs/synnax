// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const pythonClientNav: PageNavNode = {
  key: "python-client",
  name: "Python Client",
  children: [
    {
      key: "/python-client/get-started",
      href: "/python-client/get-started",
      name: "Get Started",
    },
    {
      key: "/python-client/channels",
      href: "/python-client/channels",
      name: "Channels",
    },
    {
      key: "/python-client/named-ranges",
      href: "/python-client/named-ranges",
      name: "Named Ranges",
    },
    {
      key: "/python-client/read-telemetry",
      href: "/python-client/read-telemetry",
      name: "Read Telemetry",
    },
    {
      key: "/python-client/write-telemetry",
      href: "/python-client/write-telemetry",
      name: "Write Telemetry",
    },
    {
      key: "/python-client/stream-telemetry",
      href: "/python-client/stream-telemetry",
      name: "Stream Telemetry",
    },
    {
      key: "/python-client/examples",
      href: "/python-client/examples",
      name: "Examples",
    },
    {
      key: "/python-client/troubleshooting",
      href: "/python-client/troubleshooting",
      name: "Troubleshooting",
    },
  ],
};
