// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/FavoriteButton.css";

import { Button, Icon, Text } from "@synnaxlabs/pluto";

import { CSS } from "@/css";

export interface FavoriteButtonProps extends Omit<Button.ButtonProps, "onClick"> {
  isFavorite: boolean;
  onFavorite: Button.ButtonProps["onClick"];
}

export const FavoriteButton = ({
  isFavorite,
  onFavorite,
  ghost,
  ...rest
}: FavoriteButtonProps) => (
  <Button.Button
    className={CSS(CSS.B("favorite-button"), isFavorite && CSS.M("favorite"))}
    onClick={onFavorite}
    tooltip={
      <Text.Text level="small" color={10}>
        {isFavorite ? "Remove from" : "Add to"} favorites
      </Text.Text>
    }
    variant="text"
    ghost={isFavorite ? false : ghost}
    {...rest}
  >
    {isFavorite ? <Icon.StarFilled /> : <Icon.StarOutlined />}
  </Button.Button>
);
