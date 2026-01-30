// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const CONCEPTS_NAV: PageNavNode = {
  key: "concepts",
  name: "Concepts",
  children: [
    {
      key: "/reference/control/arc/concepts/sequences-and-stages",
      href: "/reference/control/arc/concepts/sequences-and-stages",
      name: "Sequences and Stages",
    },
    {
      key: "/reference/control/arc/concepts/reactive-execution",
      href: "/reference/control/arc/concepts/reactive-execution",
      name: "Reactive Execution",
    },
    {
      key: "/reference/control/arc/concepts/channels-and-series",
      href: "/reference/control/arc/concepts/channels-and-series",
      name: "Channels and Series",
    },
    {
      key: "/reference/control/arc/concepts/stateful-variables",
      href: "/reference/control/arc/concepts/stateful-variables",
      name: "Stateful Variables",
    },
  ],
};
