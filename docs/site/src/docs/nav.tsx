// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { acquireNav } from "../pages/acquire/nav";
import { analyzeNav } from "../pages/analyze/nav";
import { conceptsNav } from "../pages/concepts/nav";
import { deployNav } from "../pages/deploy/nav";
import { referenceNav } from "../pages/reference/nav";
import { visualizeNav } from "../pages/visualize/nav";

export const pages = [
  {
    name: "Get Started",
    key: "/",
    href: "/",
  },
  conceptsNav,
  deployNav,
  acquireNav,
  analyzeNav,
  visualizeNav,
  referenceNav,
  // rfcNav,
];
