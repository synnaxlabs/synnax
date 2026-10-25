// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";

export const CORE_NAV: PageNavNode = {
  key: "core",
  name: "Core",
  children: [
    {
      href: "/reference/core/quick-start",
      name: "Quick start",
    },
    {
      href: "/reference/core/installation",
      name: "Installation",
    },
    {
      href: "/reference/core/production",
      name: "Production",
    },
    {
      href: "/reference/core/installing-certificates",
      name: "Installing certificates",
    },
    {
      href: "/reference/core/requirements",
      name: "Requirements",
    },
    {
      href: "/reference/core/cli-reference",
      name: "CLI reference",
    },
    {
      href: "/reference/core/systemd-service",
      name: "systemd service",
    },
  ],
};
