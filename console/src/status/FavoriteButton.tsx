// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { type Button } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { FavoriteButton as BaseFavoriteButton } from "@/components";
import { useSelectIsFavorite } from "@/status/selectors";
import { toggleFavorite } from "@/status/slice";

export interface FavoriteButtonProps extends Button.ButtonProps {
  statusKey: status.Key;
}

export const FavoriteButton = ({ statusKey, ...rest }: FavoriteButtonProps) => {
  const dispatch = useDispatch();
  const isFavorite = useSelectIsFavorite(statusKey);

  const handleFavorite = () => {
    dispatch(toggleFavorite({ key: statusKey }));
  };

  return (
    <BaseFavoriteButton isFavorite={isFavorite} onFavorite={handleFavorite} {...rest} />
  );
};
