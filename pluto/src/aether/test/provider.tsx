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

/**
 * Creates a test provider for use with renderHook(). Makes Aether.use() and
 * Aether.useUnidirectional() work without a real worker thread.
 *
 * @example
 * ```typescript
 * import { renderHook } from "@testing-library/react";
 * import { createTestProvider } from "@/aether/test";
 * import { button } from "@/vis/button/aether";
 * import { Button } from "@/vis/button";
 *
 * const TestProvider = createTestProvider({
 *   [button.Button.TYPE]: button.Button,
 * });
 *
 * const { result } = renderHook(() => Button.use({
 *   aetherKey: "test",
 *   sink: mockSink,
 *   mode: "fire",
 * }), { wrapper: TestProvider });
 *
 * result.current.onMouseDown();
 * ```
 *
 * @param registry - Component registry mapping type strings to component classes
 * @returns A React component that can be used as a wrapper for renderHook
 */
export const createProvider = (
  registry: aether.ComponentRegistry,
): FC<PropsWithChildren> => {
  const TestProvider: FC<PropsWithChildren> = ({ children }) => {
    const worker = useMemo(() => {
      const [a, b] = createMockWorkers();
      aether.render({ comms: a.route("test"), registry });
      return b.route<MainMessage, AetherMessage>("test");
    }, []);

    return (
      <Provider worker={worker} workerKey="test">
        {children}
      </Provider>
    );
  };

  return TestProvider;
};
