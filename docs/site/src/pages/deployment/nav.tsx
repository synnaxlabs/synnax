// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const deploymentNav: PageNavNode = {
  key: "deployment",
  name: "Deployment",
  children: [
    {
      key: "/deployment/get-started",
      href: "/deployment/get-started",
      name: "Get Started",
    },
    {
      key: "/deployment/requirements",
      href: "/deployment/requirements",
      name: "Requirements",
    },
    {
      key: "/deployment/cli-reference",
      href: "/deployment/cli-reference",
      name: "CLI Reference",
    },
    {
      key: "/deployment/systemd-service",
      href: "/deployment/systemd-service",
      name: "Systemd Service",
    },
  ],
};
