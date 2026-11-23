// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type {
  DiscoverProps,
  Kind,
  Params,
} from "@/feature/mqtt/task/discover/Discover";
import { Modals } from "@/platform/modals";
import { Panel } from "@/platform/panel";

/**
 * Creates a hook that opens a dialog listing what the broker publishes, so the user can
 * pick values. The dialog resolves with what the kind creates from the picks, or null
 * when the user closes it. The dialog loads its content when it first opens.
 */
export const createDiscoverModal = <E,>(
  kind: Kind<E>,
): Modals.PromptHook<Params, E[]> => {
  const Discover = Panel.lazyComponent<"Discover", DiscoverProps<E>>(
    () => import("@/feature/mqtt/task/discover/Discover"),
    "Discover",
  );
  const Content = (props: Modals.ContentProps<Params, E[]>) => (
    <Discover {...props} kind={kind} />
  );
  return Modals.createPrompt<E[], Params>(Content);
};
