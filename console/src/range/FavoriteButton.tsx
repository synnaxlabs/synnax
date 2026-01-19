// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { useDispatch } from "react-redux";

import {
  FavoriteButton as Base,
  type FavoriteButtonProps as BaseProps,
} from "@/components";
import { useSelect } from "@/range/selectors";
import { add, remove } from "@/range/slice";
import { fromClientRange } from "@/range/translate";

export interface FavoriteButtonProps extends Omit<
  BaseProps,
  "isFavorite" | "onFavorite"
> {
  range: ranger.Range;
}

export const FavoriteButton = ({ range, ...rest }: FavoriteButtonProps) => {
  const sliceRange = useSelect(range.key);
  const dispatch = useDispatch();
  const isFavorite = sliceRange != null;
  const handleFavorite = () => {
    if (!isFavorite) dispatch(add({ ranges: fromClientRange(range) }));
    else dispatch(remove({ keys: [range.key] }));
  };
  return <Base {...rest} isFavorite={isFavorite} onFavorite={handleFavorite} />;
};
