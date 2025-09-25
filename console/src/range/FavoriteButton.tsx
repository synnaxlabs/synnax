// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/FavoriteButton.css";

import { type ranger } from "@synnaxlabs/client";
import { Button, Icon, Text } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { useSelect } from "@/range/selectors";
import { add, remove } from "@/range/slice";
import { fromClientRange } from "@/range/translate";

export interface FavoriteButtonProps extends Button.ButtonProps {
  range: ranger.Range;
}

export const FavoriteButton = ({ range, ghost, ...rest }: FavoriteButtonProps) => {
  const sliceRange = useSelect(range.key);
  const dispatch = useDispatch();
  const starred = sliceRange != null;
  const handleStar = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    if (!starred) dispatch(add({ ranges: fromClientRange(range) }));
    else dispatch(remove({ keys: [range.key] }));
  };
  return (
    <Button.Button
      className={CSS(CSS.BE("range", "favorite-button"), starred && CSS.M("favorite"))}
      onClick={handleStar}
      tooltip={
        <Text.Text level="small" color={10}>
          {starred ? "Remove from" : "Add to"} Favorites
        </Text.Text>
      }
      variant="text"
      ghost={starred ? false : ghost}
      {...rest}
    >
      {sliceRange != null ? <Icon.StarFilled /> : <Icon.StarOutlined />}
    </Button.Button>
  );
};
