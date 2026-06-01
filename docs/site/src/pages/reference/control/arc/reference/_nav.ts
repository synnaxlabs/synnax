// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";
import { STANDARD_LIBRARY_NAV } from "@/pages/reference/control/arc/reference/standard-library/_nav";

export const REFERENCE_NAV: PageNavNode = {
  key: "reference",
  href: "/reference/control/arc/reference",
  name: "Language Reference",
  children: [
    {
      key: "/reference/control/arc/reference/syntax",
      href: "/reference/control/arc/reference/syntax",
      name: "Syntax",
    },
    {
      key: "/reference/control/arc/reference/types",
      href: "/reference/control/arc/reference/types",
      name: "Types",
    },
    {
      key: "/reference/control/arc/reference/variables",
      href: "/reference/control/arc/reference/variables",
      name: "Variables",
    },
    {
      key: "/reference/control/arc/reference/operators",
      href: "/reference/control/arc/reference/operators",
      name: "Operators",
    },
    {
      key: "/reference/control/arc/reference/flow",
      href: "/reference/control/arc/reference/flow",
      name: "Flow",
    },
    {
      key: "/reference/control/arc/reference/sequences",
      href: "/reference/control/arc/reference/sequences",
      name: "Sequences",
    },
    {
      key: "/reference/control/arc/reference/stages",
      href: "/reference/control/arc/reference/stages",
      name: "Stages",
    },
    {
      key: "/reference/control/arc/reference/functions",
      href: "/reference/control/arc/reference/functions",
      name: "Functions",
    },
    {
      key: "/reference/control/arc/reference/statements",
      href: "/reference/control/arc/reference/statements",
      name: "Statements",
    },
    {
      key: "/reference/control/arc/reference/loops",
      href: "/reference/control/arc/reference/loops",
      name: "Loops",
    },
    STANDARD_LIBRARY_NAV,
    {
      key: "/reference/control/arc/reference/errors",
      href: "/reference/control/arc/reference/errors",
      name: "Errors",
    },
  ],
};
