// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type PropsWithChildren, useState } from "react";

import { aether } from "@/aether/aether";
import { Provider } from "@/aether/main";

/** Comms that build their worker tree on first use rather than on construction. The
 * Store only touches them when it connects, which happens on commit, so a render React
 * discards allocates a closure and nothing else. */
const lazyComms = (
  registry: aether.ComponentRegistry,
  onRoot?: (root: aether.Root) => void,
): aether.MainComms => {
  let comms: aether.MainComms | null = null;
  const ensure = (): aether.MainComms => {
    if (comms == null) {
      const [workerSide, mainSide] = aether.createMockPair();
      const root = aether.render({ worker: workerSide, registry });
      onRoot?.(root);
      comms = mainSide;
    }
    return comms;
  };
  return {
    send: (value, transfer) => ensure().send(value, transfer),
    handle: (handler) => ensure().handle(handler),
  };
};

/**
 * Creates a provider that runs the worker tree on the main thread.
 * @param onRoot - Called with the root of the worker tree when the provider first
 * talks to it, so a spec can read component state with `root.findChildAtPath`.
 */
export const createAetherProvider = (
  registry: aether.ComponentRegistry,
  onRoot?: (root: aether.Root) => void,
): FC<PropsWithChildren> => {
  const TestProvider: FC<PropsWithChildren> = ({ children }) => {
    // State rather than a memo: React may drop a memo and recompute it, and one comms
    // per mount is what keeps one test's worker tree out of the next test's.
    const [comms] = useState(() => lazyComms(registry, onRoot));
    return <Provider worker={comms}>{children}</Provider>;
  };

  return TestProvider;
};
