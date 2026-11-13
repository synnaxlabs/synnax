// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Settings } from "@/app/settings";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";

const client = createTestClient();

const settingsTabs = (root: panel.Node): panel.Tab[] => {
  if (root.variant !== "leaf") throw new Error("expected a single-leaf root");
  return root.tabs.filter((tab) => tab.variant === "view" && tab.type === "settings");
};

describe("Settings.useOpen", () => {
  it("should open one Settings tab in the panel, however often it runs", async () => {
    const existing = await createServerPanel(client, {
      variant: "leaf",
      tabs: [{ variant: "view", key: uuid.create(), type: "other" }],
    });
    const { wrapper } = await createPanelWrapper({ client, panelKey: existing.key });
    await primePanel(wrapper, existing.key);
    const { result } = renderHook(() => Settings.useOpen(), { wrapper });
    await act(async () => result.current());
    await waitFor(async () => {
      const doc = await client.panels.retrieve(existing.key);
      expect(settingsTabs(doc.root)).toHaveLength(1);
    });
    await act(async () => result.current());
    const doc = await client.panels.retrieve(existing.key);
    expect(settingsTabs(doc.root)).toHaveLength(1);
  });
});
