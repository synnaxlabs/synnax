import "@/range/FavoriteButton.css";

import { type ranger } from "@synnaxlabs/client";
import { Button, Icon, Text, Tooltip } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { fromClientRange } from "@/range/ContextMenu";
import { useSelect } from "@/range/selectors";
import { add, remove } from "@/range/slice";

export interface FavoriteButtonProps {
  range: ranger.Range;
}

export const FavoriteButton = ({ range }: FavoriteButtonProps) => {
  const sliceRange = useSelect(range.key);
  const dispatch = useDispatch();
  const starred = sliceRange != null;
  const handleStar = () => {
    if (!starred) dispatch(add({ ranges: fromClientRange(range) }));
    else dispatch(remove({ keys: [range.key] }));
  };
  return (
    <Tooltip.Dialog>
      <Text.Text level="small" shade={10}>
        {starred ? "Remove from" : "Add to"} Workspace Favorites
      </Text.Text>
      <Button.Icon
        className={CSS(
          CSS.BE("range", "favorite-button"),
          starred && CSS.M("favorite"),
        )}
        onClick={(e) => {
          e.stopPropagation();
          handleStar();
        }}
        size="small"
      >
        {sliceRange != null ? <Icon.StarFilled /> : <Icon.StarOutlined />}
      </Button.Icon>
    </Tooltip.Dialog>
  );
};
