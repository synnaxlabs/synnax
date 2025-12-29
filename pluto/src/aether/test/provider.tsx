// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createMockWorkers } from "@synnaxlabs/x";
import { type FC, type PropsWithChildren, useMemo } from "react";

import { aether } from "@/aether/aether";
import { Provider } from "@/aether/main";
import { type AetherMessage, type MainMessage } from "@/aether/message";

export const createProvider = (
  registry: aether.ComponentRegistry,
): FC<PropsWithChildren> => {
  const TestProvider: FC<PropsWithChildren> = ({ children }) => {
    const worker = useMemo(() => {
      const [worker, main] = createMockWorkers();
      aether.render({ comms: worker.route("test"), registry });
      return main.route<MainMessage, AetherMessage>("test");
    }, []);

    return (
      <Provider worker={worker} workerKey="test">
        {children}
      </Provider>
    );
  };

  return TestProvider;
};
