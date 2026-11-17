// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Explorer } from "@/feature/status/explorer";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { Session } from "@/session";

const client = createTestClient();

const OTHER = "other";

/** Opens the explorer from a panel whose given tab has focus. */
const openFrom = async (tabs: panel.Tab[], focused: panel.TabKey) => {
  const { key } = await createServerPanel(client, { variant: "leaf", tabs });
  const { wrapper, store } = await createPanelWrapper({
    client,
    panelKey: key,
    tabKey: focused,
  });
  await primePanel(wrapper, key);
  const { result } = renderHook(() => Explorer.useOpenTab(), { wrapper });
  await act(async () => result.current());
  return { key, store };
};

describe("Explorer.useOpenTab", () => {
  it("should focus the explorer tab already open in the panel", async () => {
    const [explorer, other] = [uuid.create(), uuid.create()];
    const { key, store } = await openFrom(
      [
        { variant: "view", key: explorer, type: Explorer.TAB_TYPE },
        { variant: "view", key: other, type: OTHER },
      ],
      other,
    );
    await waitFor(() =>
      expect(Session.Panel.selectSelectedTabs(store.getState(), key)[0]).toBe(explorer),
    );
    const { root } = await client.panels.retrieve(key);
    expect(panel.findTabLeaf(root, explorer)?.tabs).toHaveLength(2);
  });

  it("should open an explorer tab when the panel has none", async () => {
    const other = uuid.create();
    const { key } = await openFrom(
      [{ variant: "view", key: other, type: OTHER }],
      other,
    );
    await waitFor(async () => {
      const { root } = await client.panels.retrieve(key);
      expect(panel.findTabByType(root, Explorer.TAB_TYPE)).toBeDefined();
    });
  });
});
