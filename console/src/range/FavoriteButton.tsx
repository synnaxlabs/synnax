import "@/range/FavoriteButton.css";

import { type ranger } from "@synnaxlabs/client";
import { Button, Icon, Text } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { fromClientRange } from "@/range/ContextMenu";
import { useSelect } from "@/range/selectors";
import { add, remove } from "@/range/slice";

export interface FavoriteButtonProps extends Button.ButtonProps {
  range: ranger.Range;
}

export const FavoriteButton = ({ range, ghost, ...rest }: FavoriteButtonProps) => {
  const sliceRange = useSelect(range.key);
  const dispatch = useDispatch();
  const starred = sliceRange != null;
  const handleStar = () => {
    if (!starred) dispatch(add({ ranges: fromClientRange(range) }));
    else dispatch(remove({ keys: [range.key] }));
  };
  return (
    <Button.Button
      className={CSS(CSS.BE("range", "favorite-button"), starred && CSS.M("favorite"))}
      onClick={(e) => {
        e.stopPropagation();
        handleStar();
      }}
      tooltip={
        <Text.Text level="small" color={10}>
          {starred ? "Remove from" : "Add to"} Workspace Favorites
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
