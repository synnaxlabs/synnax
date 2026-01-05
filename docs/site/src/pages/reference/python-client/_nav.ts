// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const PYTHON_CLIENT_NAV: PageNavNode = {
  key: "python-client",
  name: "Python Client",
  children: [
    {
      key: "/reference/python-client/get-started",
      href: "/reference/python-client/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/python-client/channels",
      href: "/reference/python-client/channels",
      name: "Channels",
    },
    {
      key: "/reference/python-client/ranges",
      href: "/reference/python-client/ranges",
      name: "Ranges",
    },
    {
      key: "/reference/python-client/read-data",
      href: "/reference/python-client/read-data",
      name: "Read Data",
    },
    {
      key: "/reference/python-client/write-data",
      href: "/reference/python-client/write-data",
      name: "Write Data",
    },
    {
      key: "/reference/python-client/stream-data",
      href: "/reference/python-client/stream-data",
      name: "Stream Data",
    },
    {
      key: "/reference/python-client/delete-data",
      href: "/reference/python-client/delete-data",
      name: "Delete Data",
    },
    {
      key: "/reference/python-client/series-and-frames",
      href: "/reference/python-client/series-and-frames",
      name: "Series and Frames",
    },
    {
      key: "/reference/python-client/examples",
      href: "/reference/python-client/examples",
      name: "Examples",
    },
    {
      key: "/reference/python-client/troubleshooting",
      href: "/reference/python-client/troubleshooting",
      name: "Troubleshooting",
    },
    {
      key: "/reference/python-client/device-driver",
      href: "/reference/python-client/device-driver",
      name: "Build a Device Driver",
    },
  ],
};
