// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";

import { type Modals } from "@/platform/modals";
import { type ModalOpenerHandle, renderModalOpener } from "@/platform/modals/testutil";

export interface OpenModalOptions<P> {
  client?: Synnax | null;
  params?: P;
}

/**
 * Opens the given modal-opener hook inside the full console provider stack with a
 * mounted modal stack, and returns the render result plus the console store. Pass a real
 * client to exercise the enabled/save path, or omit it (null) to exercise the no-cluster
 * branch.
 */
export const openModal = async <P>(
  useOpen: () => Modals.Opener<P>,
  { client = null, params }: OpenModalOptions<P> = {},
): Promise<ModalOpenerHandle<void>> =>
  await renderModalOpener(useOpen as () => (params?: P) => void, [params], {
    client,
  });
