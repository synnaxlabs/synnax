// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";

export const CONCEPTS_NAV: PageNavNode = {
  key: "arc-concepts",
  name: "Concepts",
  children: [
    {
      href: "/reference/control/arc/concepts/control-authority",
      name: "Control authority",
    },
    {
      href: "/reference/control/arc/concepts/reactive-execution",
      name: "Reactive execution",
    },
    {
      href: "/reference/control/arc/concepts/sequences-and-stages",
      name: "Sequences and stages",
    },
    {
      href: "/reference/control/arc/concepts/channels-and-series",
      name: "Channels and series",
    },
  ],
};
