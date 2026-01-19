// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const CORE_NAV: PageNavNode = {
  key: "core",
  name: "Core",
  children: [
    {
      key: "/reference/core/quick-start",
      href: "/reference/core/quick-start",
      name: "Quick Start",
    },
    {
      key: "/reference/core/installation",
      href: "/reference/core/installation",
      name: "Installation",
    },
    {
      key: "/reference/core/production",
      href: "/reference/core/production",
      name: "Production",
    },
    {
      key: "/reference/core/requirements",
      href: "/reference/core/requirements",
      name: "Requirements",
    },
    {
      key: "/reference/core/cli-reference",
      href: "/reference/core/cli-reference",
      name: "CLI Reference",
    },
    {
      key: "/reference/core/systemd-service",
      href: "/reference/core/systemd-service",
      name: "systemd Service",
    },
  ],
};
