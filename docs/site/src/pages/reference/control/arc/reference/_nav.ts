// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const REFERENCE_NAV: PageNavNode = {
  key: "reference",
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
      key: "/reference/control/arc/reference/operators",
      href: "/reference/control/arc/reference/operators",
      name: "Operators",
    },
    {
      key: "/reference/control/arc/reference/functions",
      href: "/reference/control/arc/reference/functions",
      name: "Functions",
    },
    {
      key: "/reference/control/arc/reference/sequences",
      href: "/reference/control/arc/reference/sequences",
      name: "Sequences",
    },
    {
      key: "/reference/control/arc/reference/built-ins",
      href: "/reference/control/arc/reference/built-ins",
      name: "Built-In Functions",
    },
    {
      key: "/reference/control/arc/reference/errors",
      href: "/reference/control/arc/reference/errors",
      name: "Errors",
    },
  ],
};
