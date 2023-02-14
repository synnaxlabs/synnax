// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PageNavLeaf } from "@/components/PageNav";

export const pythonClientNav: PageNavLeaf = {
  key: "python-client",
  name: "Python Client",
  children: [
    {
      key: "/python-client/get-started",
      name: "Get Started",
      url: "/python-client/get-started",
    },
  ],
};
