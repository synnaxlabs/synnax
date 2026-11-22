// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { ComponentType } from "react";

import { Settings } from "@/app/settings";
import { Secret } from "@/feature/secret";
import { Tree } from "@/platform/tree";
import { createConsoleWrapper, renderLazy } from "@/testutil";

/** Renders the Settings tab with the Console providers, as the given client. */
export const renderSettings = async (
  as: Synnax = createTestClient(),
): Promise<{ wrapper: ComponentType }> => {
  const { Content } = Settings.TABS.settings;
  const { wrapper } = await createConsoleWrapper({ client: as });
  await renderLazy(
    <Tree.Provider items={Secret.TREE_ITEMS}>
      <Content />
    </Tree.Provider>,
    { wrapper },
  );
  return { wrapper };
};
