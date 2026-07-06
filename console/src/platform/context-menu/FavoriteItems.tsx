// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Menu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export interface FavoriteItemsProps {
  anyFavorited: boolean;
  anyNotFavorited: boolean;
  onFavorite: () => void;
  onUnfavorite: () => void;
}

export const FavoriteItems = ({
  anyFavorited,
  anyNotFavorited,
  onFavorite,
  onUnfavorite,
}: FavoriteItemsProps): ReactElement => (
  <>
    {anyNotFavorited && (
      <Menu.Item itemKey="favorite" onClick={onFavorite}>
        <Icon.StarFilled />
        Favorite
      </Menu.Item>
    )}
    {anyFavorited && (
      <Menu.Item itemKey="unfavorite" onClick={onUnfavorite}>
        <Icon.StarOutlined />
        Unfavorite
      </Menu.Item>
    )}
  </>
);
