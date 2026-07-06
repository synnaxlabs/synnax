// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ContextMenu } from "@/platform/context-menu";
import { renderWithConsole } from "@/testutil";

const render = async (props: Partial<ContextMenu.FavoriteItemsProps> = {}) => {
  const onFavorite = vi.fn();
  const onUnfavorite = vi.fn();
  await renderWithConsole(
    <Menu.Menu>
      <ContextMenu.FavoriteItems
        anyFavorited={false}
        anyNotFavorited={false}
        onFavorite={onFavorite}
        onUnfavorite={onUnfavorite}
        {...props}
      />
    </Menu.Menu>,
  );
  return { onFavorite, onUnfavorite };
};

describe("ContextMenu.FavoriteItems", () => {
  it("shows only Favorite when some items are not favorited", async () => {
    const { onFavorite } = await render({ anyNotFavorited: true });
    expect(screen.queryByText("Unfavorite")).toBeNull();
    fireEvent.click(screen.getByText("Favorite"));
    expect(onFavorite).toHaveBeenCalledTimes(1);
  });

  it("shows only Unfavorite when some items are favorited", async () => {
    const { onUnfavorite } = await render({ anyFavorited: true });
    expect(screen.queryByText("Favorite")).toBeNull();
    fireEvent.click(screen.getByText("Unfavorite"));
    expect(onUnfavorite).toHaveBeenCalledTimes(1);
  });

  it("shows both actions for a mixed selection", async () => {
    await render({ anyFavorited: true, anyNotFavorited: true });
    expect(screen.getByText("Favorite")).toBeTruthy();
    expect(screen.getByText("Unfavorite")).toBeTruthy();
  });

  it("shows neither action when nothing is selectable", async () => {
    await render();
    expect(screen.queryByText("Favorite")).toBeNull();
    expect(screen.queryByText("Unfavorite")).toBeNull();
  });
});
