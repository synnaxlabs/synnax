// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type PropsWithChildren } from "react";

import { aether } from "@/aether/aether";
import { Provider } from "@/aether/main";

export const createProvider = (
  registry: aether.ComponentRegistry,
): FC<PropsWithChildren> => {
  // Built once per provider rather than per render: constructing the worker tree is
  // resource acquisition, and a render React discards would otherwise strand one.
  const [workerSide, mainSide] = aether.createMockPair();
  aether.render({ worker: workerSide, registry });

  const TestProvider: FC<PropsWithChildren> = ({ children }) => (
    <Provider worker={mainSide}>{children}</Provider>
  );

  return TestProvider;
};
