// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";
import { STANDARD_LIBRARY_NAV } from "@/pages/reference/control/arc/reference/standard-library/_nav";

export const REFERENCE_NAV: PageNavNode = {
  href: "/reference/control/arc/reference",
  name: "Language reference",
  children: [
    {
      href: "/reference/control/arc/reference/syntax",
      name: "Syntax",
    },
    {
      href: "/reference/control/arc/reference/types",
      name: "Types",
    },
    {
      href: "/reference/control/arc/reference/variables",
      name: "Variables",
    },
    {
      href: "/reference/control/arc/reference/operators",
      name: "Operators",
    },
    {
      href: "/reference/control/arc/reference/flow",
      name: "Flow",
    },
    {
      href: "/reference/control/arc/reference/sequences",
      name: "Sequences",
    },
    {
      href: "/reference/control/arc/reference/stages",
      name: "Stages",
    },
    {
      href: "/reference/control/arc/reference/functions",
      name: "Functions",
    },
    {
      href: "/reference/control/arc/reference/statements",
      name: "Statements",
    },
    {
      href: "/reference/control/arc/reference/loops",
      name: "Loops",
    },
    STANDARD_LIBRARY_NAV,
    {
      href: "/reference/control/arc/reference/errors",
      name: "Errors",
    },
  ],
};
