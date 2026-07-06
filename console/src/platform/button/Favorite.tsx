// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/button/Favorite.css";

import { Button, Icon } from "@synnaxlabs/pluto";

import { CSS } from "@/platform/css";

export interface FavoriteProps extends Omit<Button.ButtonProps, "onClick"> {
  isFavorite: boolean;
  onFavorite: Button.ButtonProps["onClick"];
}

export const Favorite = ({ isFavorite, onFavorite, ghost, ...rest }: FavoriteProps) => (
  <Button.Button
    className={CSS(CSS.B("favorite-button"), isFavorite && CSS.M("favorite"))}
    onClick={onFavorite}
    tooltip={`${isFavorite ? "Unfavorite" : "Favorite"}`}
    variant="text"
    ghost={isFavorite ? false : ghost}
    {...rest}
  >
    {isFavorite ? <Icon.StarFilled /> : <Icon.StarOutlined />}
  </Button.Button>
);
