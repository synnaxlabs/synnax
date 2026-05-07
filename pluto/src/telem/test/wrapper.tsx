// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { aether } from "@synnaxlabs/charon/aether/runtime";
import { aetherTest } from "@synnaxlabs/charon/aether/test";
import { status } from "@synnaxlabs/charon/status/aether";
import { type Synnax as SynnaxClient } from "@synnaxlabs/client";
import { type FC, type PropsWithChildren } from "react";

import { Alamos } from "@/alamos";
import { alamos } from "@/alamos/aether";
import { Status } from "@/status";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import { Telem } from "@/telem";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/test";

export interface CreateTestWrapperOptions {
  registry: aether.ComponentRegistry;
  client?: SynnaxClient | null;
  telemFactories?: telem.Factory[];
}

export const createTestWrapper = (
  options: CreateTestWrapperOptions,
): FC<PropsWithChildren> => {
  const { registry, client = null, telemFactories = [] } = options;

  const TelemProvider = telem.createProvider(
    () =>
      new telem.CompoundFactory([
        new telemTest.TestFactory(),
        new telem.NoopFactory(),
        ...telemFactories,
      ]),
  );

  const AetherProvider = aetherTest.createProvider({
    ...registry,
    [telem.PROVIDER_TYPE]: TelemProvider,
    ...synnax.REGISTRY,
    ...status.REGISTRY,
    ...alamos.REGISTRY,
  });

  const TestWrapper: FC<PropsWithChildren> = ({ children }) => (
    <AetherProvider>
      <Status.Aggregator>
        <Alamos.Provider>
          <Synnax.TestProvider client={client}>
            <Telem.Provider>{children}</Telem.Provider>
          </Synnax.TestProvider>
        </Alamos.Provider>
      </Status.Aggregator>
    </AetherProvider>
  );

  return TestWrapper;
};

// DONE: accepts custom telem factories via telemFactories option.
