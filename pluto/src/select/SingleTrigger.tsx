import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { type Icon } from "@/icon";
import { List } from "@/list";
import { useContext, useSelection } from "@/select/Frame";
import { canDrop } from "@/select/MultipleTrigger";

export interface SingleTriggerEntry<K extends record.Key> extends record.KeyedNamed<K> {
  icon?: Icon.ReactElement;
}

export interface SingleTriggerProps {
  haulType?: string;
  placeholder?: string;
  icon?: Icon.ReactElement;
}

export const SingleTrigger = <K extends record.Key>({
  haulType = "",
  placeholder,
  icon: baseIcon,
}: SingleTriggerProps) => {
  const allSelected = useSelection<K>();
  const { onSelect } = useContext<K>();
  const [selected] = allSelected;
  const item = List.useItem<K, SingleTriggerEntry<K>>(selected);
  const { name, icon } = item ?? {};
  const dropProps = Haul.useDrop({
    type: "Ranger.SelectSingle",
    canDrop: useCallback(
      (hauled) => canDrop(hauled, haulType, allSelected),
      [haulType, allSelected],
    ),
    onDrop: Haul.useFilterByTypeCallback(
      haulType,
      ({ items }) => {
        if (items.length !== 0) onSelect(items[0].key as K);
        return items;
      },
      [onSelect],
    ),
  });
  const dragging = Haul.useDraggingState();
  return (
    <Dialog.Trigger
      variant="outlined"
      startIcon={icon ?? baseIcon}
      className={CSS(CSS.dropRegion(canDrop(dragging, haulType, allSelected)))}
      {...dropProps}
    >
      {name ?? placeholder}
    </Dialog.Trigger>
  );
};
