// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";

import { Button } from "@/platform/button";
import { Session } from "@/session";

export interface FavoriteButtonProps extends Omit<
  Button.FavoriteProps,
  "isFavorite" | "onFavorite"
> {
  range: ranger.Range;
}

export const FavoriteButton = ({ range, ...rest }: FavoriteButtonProps) => {
  const sliceRange = Session.Range.useSelectState(range.key);
  const dispatch = Session.useDispatch();
  const isFavorite = sliceRange != null;
  const handleFavorite = () => {
    if (!isFavorite)
      Session.Range.fromClient(range).forEach((r) => dispatch(Session.Range.add(r)));
    else dispatch(Session.Range.remove({ keys: [range.key] }));
  };
  return (
    <Button.Favorite {...rest} isFavorite={isFavorite} onFavorite={handleFavorite} />
  );
};
