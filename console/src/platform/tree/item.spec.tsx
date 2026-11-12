// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Tree } from "@/platform/tree";

const createResource = (key: string, name: string): resource.Resource => {
  const id: resource.ID = { type: "group", key };
  return { key: resource.idToString(id), id, name };
};

describe("Tree.createItem", () => {
  it("returns the icon that the item shows for each resource", () => {
    const named = <Icon.Group />;
    const other = <Icon.Channel />;
    const item = Tree.createItem({
      icon: (r) => (r.name === "named" ? named : other),
    });
    expect(item.icon(createResource("a", "named"))?.type).toBe(named.type);
    expect(item.icon(createResource("b", "other"))?.type).toBe(other.type);
  });

  it("returns no icon for an item without one", () => {
    expect(Tree.createItem({}).icon(createResource("a", "a"))).toBeUndefined();
  });

  it("hands its children the handler that a double click runs", () => {
    const onSelect = vi.fn();
    const item = Tree.createItem({ useOnSelect: () => onSelect });
    const target = createResource("a", "a");
    render(
      <item.OnSelect>
        {(select) => <button onClick={() => select(target)}>open</button>}
      </item.OnSelect>,
    );
    fireEvent.click(screen.getByText("open"));
    expect(onSelect).toHaveBeenCalledExactlyOnceWith(target);
  });
});
