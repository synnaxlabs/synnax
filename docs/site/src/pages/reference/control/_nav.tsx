// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";
import { embeddedSequencesNav } from "@/pages/reference/control/embedded/_nav";
import { pythonSequencesNav } from "@/pages/reference/control/python/_nav";

export const controlNav: PageNavNode = {
  key: "control",
  name: "Control",
  children: [
    {
      key: "/reference/control/get-started",
      href: "/reference/control/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/control/control-authority",
      href: "/reference/control/control-authority",
      name: "Control Authority",
    },
    pythonSequencesNav,
    embeddedSequencesNav,
  ],
};
