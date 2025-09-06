// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";
import { ANALYST_NAV } from "@/pages/guides/analyst/_nav";
import { COMPARISON_NAV } from "@/pages/guides/comparison/_nav";
import { GET_STARTED_NAV } from "@/pages/guides/get-started/_nav";
import { OPERATIONS_NAV } from "@/pages/guides/operations/_nav";
import { SYS_ADMIN_NAV } from "@/pages/guides/sys-admin/_nav";
import { CLUSTER_NAV } from "@/pages/reference/cluster/_nav";
import { CONCEPTS_NAV } from "@/pages/reference/concepts/_nav";
import { CONSOLE_NAV } from "@/pages/reference/console/_nav";
import { CONTROL_NAV } from "@/pages/reference/control/_nav";
import { DEVICE_DRIVERS_NAV } from "@/pages/reference/device-drivers/_nav";
import { PLUTO_NAV } from "@/pages/reference/pluto/_nav";
import { PYTHON_CLIENT_NAV } from "@/pages/reference/python-client/_nav";
import { TYPESCRIPT_CLIENT_NAV } from "@/pages/reference/typescript-client/_nav";

export const REFERENCE_PAGES: PageNavNode[] = [
  { name: "Get Started", key: "/reference/", href: "/reference/" },
  CONCEPTS_NAV,
  CLUSTER_NAV,
  TYPESCRIPT_CLIENT_NAV,
  PYTHON_CLIENT_NAV,
  CONTROL_NAV,
  CONSOLE_NAV,
  DEVICE_DRIVERS_NAV,
  PLUTO_NAV,
];

export const GUIDES_PAGES: PageNavNode[] = [
  { name: "Why Synnax?", key: "/guides/", href: "/guides/" },
  GET_STARTED_NAV,
  ANALYST_NAV,
  SYS_ADMIN_NAV,
  OPERATIONS_NAV,
  COMPARISON_NAV,
];
