// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";

import { Button } from "@/primitive/button";
import { Session } from "@/session";
import { useSelectState } from "@/session/range/selectors";
import { add, remove } from "@/session/range/slice";

export interface FavoriteButtonProps extends Omit<
  Button.FavoriteProps,
  "isFavorite" | "onFavorite"
> {
  range: ranger.Range;
}

export const FavoriteButton = ({ range, ...rest }: FavoriteButtonProps) => {
  const sliceRange = useSelectState(range.key);
  const dispatch = Session.useDispatch();
  const isFavorite = sliceRange != null;
  const handleFavorite = () => {
    if (!isFavorite) Session.Range.fromClient(range).forEach((r) => dispatch(add(r)));
    else dispatch(remove({ keys: [range.key] }));
  };
  return (
    <Button.Favorite {...rest} isFavorite={isFavorite} onFavorite={handleFavorite} />
  );
};
