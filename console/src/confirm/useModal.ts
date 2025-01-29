// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { useStore } from "react-redux";

import {
  configureLayout,
  type LayoutArgs,
  type LayoutOverrides,
} from "@/confirm/Confirm";
import { Layout } from "@/layout";
import { type RootState } from "@/store";

export interface CreateModal {
  (args: LayoutArgs, layoutOverrides?: LayoutOverrides): Promise<boolean>;
}

export const useModal = (): CreateModal => {
  const placeLayout = Layout.usePlacer();
  const store = useStore<RootState>();
  return async (args, layoutOverrides) => {
    let unsubscribe: ReturnType<typeof store.subscribe> | null = null;
    return await new Promise((resolve) => {
      const layout = configureLayout(args, layoutOverrides);
      placeLayout(layout);
      unsubscribe = store.subscribe(() => {
        const l = Layout.select(store.getState(), layout.key);
        if (l == null) resolve(false);
        const args = Layout.selectArgs<LayoutArgs>(store.getState(), layout.key);
        // This means the action was unrelated to the confirmation.
        if (args != null && args.result == null) return;
        // This means that the layout was removed by a separate mechanism than
        // the user hitting 'Confirm' or 'Cancel'. We treat this as a cancellation.
        if (args == null) resolve(false);
        // Resolve with the standard result.
        else resolve(args.result as boolean);
        unsubscribe?.();
      });
    });
  };
};
