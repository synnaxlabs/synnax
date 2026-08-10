// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Input } from "@synnaxlabs/pluto";

export interface FavoriteProps extends Omit<Input.CheckboxProps, "value" | "onChange"> {
  isFavorite: boolean;
  onFavorite: () => void;
}

export const Favorite = ({
  isFavorite,
  onFavorite,
  reveal,
  ...rest
}: FavoriteProps) => (
  <Input.Checkbox
    value={isFavorite}
    onChange={onFavorite}
    checkedIcon={<Icon.StarFilled />}
    uncheckedIcon={<Icon.StarOutlined />}
    textColor={isFavorite ? "goldenrod" : undefined}
    tooltip={isFavorite ? "Unfavorite" : "Favorite"}
    aria-label="Favorite"
    reveal={isFavorite ? false : reveal}
    {...rest}
  />
);
