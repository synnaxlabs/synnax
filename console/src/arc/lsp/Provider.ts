// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { type PropsWithChildren } from "react";

import { use } from "@/arc/lsp/lsp";
import { Code } from "@/code";

export const Provider = ({ children }: PropsWithChildren) => {
  use(Synnax.use(), Code.useMonaco());
  return children;
};
