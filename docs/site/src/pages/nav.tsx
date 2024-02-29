// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clusterNav } from "@/pages/components/cluster/nav";
import { consoleNav } from "@/pages/components/console/nav";
import { pythonClientNav } from "@/pages/components/python-client/nav";
import { conceptsNav } from "@/pages/components/concepts/nav";
import { analystsNav } from "@/pages/roles/analysts/nav";

export const componentsPages = [
  {
    name: "Get Started",
    key: "/",
    href: "/",
  },
  conceptsNav,
  clusterNav,
  pythonClientNav,
  consoleNav,
];

export const rolesPages = [
  {
    name: "Get Started",
    key: "/",
    href: "/",
  },
  analystsNav,
];
