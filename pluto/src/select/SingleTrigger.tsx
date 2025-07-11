// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

export interface SingleTriggerProps extends Dialog.TriggerProps {
  haulType?: string;
  placeholder?: string;
  icon?: Icon.ReactElement;
}

export const SingleTrigger = <K extends record.Key>({
  haulType = "",
  placeholder,
  icon: baseIcon,
  disabled,
  ...rest
}: SingleTriggerProps) => {
  const allSelected = useSelection<K>();
  const { onSelect } = useContext<K>();
  const [selected] = allSelected;
  const item = List.useItem<K, SingleTriggerEntry<K>>(selected);
  const { name, icon } = item ?? {};
  const dropProps = Haul.useDrop({
    type: haulType,
    canDrop: useCallback(
      (hauled) => canDrop(hauled, haulType, allSelected, disabled),
      [haulType, allSelected, disabled],
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
      className={CSS(
        CSS.dropRegion(canDrop(dragging, haulType, allSelected, disabled)),
      )}
      {...dropProps}
      {...rest}
    >
      {name ?? placeholder}
    </Dialog.Trigger>
  );
};
