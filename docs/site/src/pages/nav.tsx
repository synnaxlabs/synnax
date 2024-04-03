// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { typescriptClientNav } from "@/pages/typescript-client/nav";
import { conceptsNav } from "./concepts/nav";
import { consoleNav } from "./console/nav";
import { deploymentNav } from "./deployment/nav";
import { guidesNav } from "./guides/nav";
import { pythonClientNav } from "./python-client/nav";

export const pages = [
  {
    name: "Get Started",
    key: "/",
    href: "/",
  },
  guidesNav,
  conceptsNav,
  deploymentNav,
  pythonClientNav,
  typescriptClientNav,
  consoleNav,
];
