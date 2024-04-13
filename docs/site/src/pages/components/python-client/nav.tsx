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
      key: "/components/python-client/get-started",
      href: "/components/python-client/get-started",
      name: "Get Started",
    },
    {
      key: "/components/python-client/channels",
      href: "/components/python-client/channels",
      name: "Channels",
    },
    {
      key: "/components/python-client/named-ranges",
      href: "/components/python-client/named-ranges",
      name: "Named Ranges",
    },
    {
      key: "/components/python-client/read-telemetry",
      href: "/components/python-client/read-telemetry",
      name: "Read Telemetry",
    },
    {
      key: "/components/python-client/write-telemetry",
      href: "/components/python-client/write-telemetry",
      name: "Write Telemetry",
    },
    {
      key: "/components/python-client/stream-telemetry",
      href: "/components/python-client/stream-telemetry",
      name: "Stream Telemetry",
    },
    {
      key: "/components/python-client/series-and-frames",
      href: "/components/python-client/series-and-frames",
      name: "Series and Frames",
    },
    {
      key: "/components/python-client/examples",
      href: "/components/python-client/examples",
      name: "Examples",
    },
    {
      key: "/components/python-client/troubleshooting",
      href: "/components/python-client/troubleshooting",
      name: "Troubleshooting",
    },
  ],
};
