// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clusterNav } from "./components/cluster/nav";
import { consoleNav } from "./components/console/nav";
import { pythonClientNav } from "./components/python-client/nav";
import { conceptsNav } from "./concepts/nav";
import { guidesNav } from "./guides/nav";

export const pages = [
  {
    name: "Get Started",
    key: "/",
    href: "/",
  },
  guidesNav,
  conceptsNav,
  clusterNav,
  pythonClientNav,
  consoleNav,
];
