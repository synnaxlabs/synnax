// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Tree } from "@/platform/tree";

describe("Provider", () => {
  const channelItem = Tree.createItem({ type: "channel", icon: <Icon.Channel /> });
  const items: Tree.Items = { channel: channelItem };

  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Tree.Provider items={items}>{children}</Tree.Provider>
  );

  it("should provide the item registry to consumers via useItems", () => {
    const { result } = renderHook(() => Tree.useItems(), { wrapper });
    expect(result.current.channel).toBe(channelItem);
  });

  it("should throw when useItems is used outside of a provider", () => {
    expect(() => renderHook(() => Tree.useItems())).toThrow();
  });
});
