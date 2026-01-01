// Copyright 2026 Synnax Labs, Inc.
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
import { staticCanDrop } from "@/select/MultipleTrigger";

export interface SingleTriggerEntry<K extends record.Key> extends record.KeyedNamed<K> {
  icon?: Icon.ReactElement;
}

export interface SingleTriggerProps extends Dialog.TriggerProps {
  haulType?: string;
  placeholder?: string;
  icon?: Icon.ReactElement;
  iconOnly?: boolean;
}

export const SingleTrigger = <K extends record.Key>({
  haulType = "",
  placeholder,
  icon: baseIcon,
  disabled,
  iconOnly = false,
  hideCaret = false,
  ...rest
}: SingleTriggerProps) => {
  const allSelected = useSelection<K>();
  const { setSelected } = useContext<K>();
  const [selected] = allSelected;
  const item = List.useItem<K, SingleTriggerEntry<K>>(selected);
  const { name, icon } = item ?? {};
  const canDrop = useCallback(
    (hauled: Haul.DraggingState) =>
      staticCanDrop(hauled, haulType, allSelected, disabled),
    [haulType, allSelected, disabled],
  );
  const dropProps = Haul.useDrop({
    type: haulType,
    canDrop,
    onDrop: Haul.useFilterByTypeCallback(
      haulType,
      ({ items }) => {
        if (items.length !== 0) setSelected([items[0].key as K]);
        return items;
      },
      [setSelected],
    ),
  });
  const dragging = Haul.useDraggingState();
  return (
    <Dialog.Trigger
      variant="outlined"
      gap="small"
      className={CSS(CSS.dropRegion(canDrop(dragging)))}
      disabled={disabled}
      {...dropProps}
      {...rest}
      hideCaret={hideCaret || iconOnly}
      textColor={name == null ? 8 : undefined}
    >
      {icon ?? baseIcon}
      {!iconOnly && (name ?? placeholder)}
    </Dialog.Trigger>
  );
};
