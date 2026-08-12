// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type PropsWithChildren, useMemo } from "react";

import { aether } from "@/aether/aether";
import { Provider } from "@/aether/main";

/** Comms that build their worker tree on first use rather than on construction. The
 * Store only touches them when it connects, which happens on commit, so a render React
 * discards allocates a closure and nothing else. */
const lazyComms = (registry: aether.ComponentRegistry): aether.MainComms => {
  let comms: aether.MainComms | null = null;
  const ensure = (): aether.MainComms => {
    if (comms == null) {
      const [workerSide, mainSide] = aether.createMockPair();
      aether.render({ worker: workerSide, registry });
      comms = mainSide;
    }
    return comms;
  };
  return {
    send: (value, transfer) => ensure().send(value, transfer),
    handle: (handler) => ensure().handle(handler),
  };
};

export const createProvider = (
  registry: aether.ComponentRegistry,
): FC<PropsWithChildren> => {
  const TestProvider: FC<PropsWithChildren> = ({ children }) => {
    // Per mount, so two concurrent mounts never share a worker tree.
    const worker = useMemo(() => lazyComms(registry), []);
    return <Provider worker={worker}>{children}</Provider>;
  };

  return TestProvider;
};
