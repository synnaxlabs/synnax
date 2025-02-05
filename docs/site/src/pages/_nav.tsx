// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { analystNav } from "@/pages/guides/analyst/_nav";
import { comparisonNav } from "@/pages/guides/comparison/_nav";
import { getStartedNav } from "@/pages/guides/get-started/_nav";
import { operationsNav } from "@/pages/guides/operations/_nav";
import { sysAdminNav } from "@/pages/guides/sys-admin/_nav";
import { clusterNav } from "@/pages/reference/cluster/_nav";
import { conceptsNav } from "@/pages/reference/concepts/_nav";
import { consoleNav } from "@/pages/reference/console/_nav";
import { controlNav } from "@/pages/reference/control/_nav";
import { deviceDriversNav } from "@/pages/reference/device-drivers/_nav";
import { pythonClientNav } from "@/pages/reference/python-client/_nav";
import { typescriptClientNav } from "@/pages/reference/typescript-client/_nav";

export const componentsPages = [
  { name: "Get Started", key: "/reference/", href: "/reference/" },
  conceptsNav,
  clusterNav,
  typescriptClientNav,
  pythonClientNav,
  controlNav,
  consoleNav,
  deviceDriversNav,
];

export const guidesPages = [
  { name: "Why Synnax?", key: "/guides/", href: "/guides/" },
  getStartedNav,
  analystNav,
  sysAdminNav,
  operationsNav,
  comparisonNav,
];
