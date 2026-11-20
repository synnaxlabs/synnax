// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul } from "@synnaxlabs/lyra/haul";
import { renderHook } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { type Mock, vi } from "vitest";

import { Window } from "@/platform/window";
import { createConsoleWrapper } from "@/testutil";

export const SOURCE: Haul.Item = { key: "source", type: "mosaic-tab" };
export const ITEM: Haul.Item = { key: "tab", type: "mosaic-tab" };

/**
 * Mounts Window.useDropOutside under a Haul provider that accepts every drop. Pass
 * `dragging` to show a drag that another window started.
 */
export const renderDropOutside = async (
  dragging?: Haul.State,
): Promise<{
  onDrop: Mock<Window.OnDropOutside>;
  haul: Haul.ContextValue | null;
}> => {
  const { wrapper: Base } = await createConsoleWrapper({ client: null });
  const Wrapper: FC<PropsWithChildren> = ({ children }) => (
    <Base>
      <Haul.Provider value={dragging} onChange={dragging && vi.fn()}>
        {children}
      </Haul.Provider>
    </Base>
  );
  Wrapper.displayName = "Wrapper";
  const onDrop = vi.fn<Window.OnDropOutside>();
  const useHarness = () => {
    Window.useDropOutside({ canDrop: () => true, onDrop });
    return Haul.useContext();
  };
  const { result } = renderHook(useHarness, { wrapper: Wrapper });
  return { onDrop, haul: result.current };
};
