// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clusterNav } from "@/pages/components/cluster/_nav";
import { conceptsNav } from "@/pages/components/concepts/_nav";
import { consoleNav } from "@/pages/components/console/_nav";
import { pythonClientNav } from "@/pages/components/python-client/_nav";
import { analystNav } from "@/pages/roles/analyst/nav";
import { sysAdminNav } from "@/pages/roles/sys-admin/nav";
import { operationsNav } from "@/pages/roles/operations/nav";
import { typescriptClientNav } from "@/pages/components/typescript-client/_nav";

export const componentsPages = [
  {
    name: "Get Started",
    key: "/",
    href: "/",
  },
  conceptsNav,
  clusterNav,
  pythonClientNav,
  typescriptClientNav,
  consoleNav,
];
export const rolesPages = [
  {
    name: "Get Started",
    key: "/",
    href: "/",
  },
  analystNav,
  sysAdminNav,
  operationsNav,
];
