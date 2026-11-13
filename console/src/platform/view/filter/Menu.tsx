// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu as Base } from "@synnaxlabs/lyra/menu";
import { location } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { useContext } from "@/platform/view/context";
import { drop, find, put } from "@/platform/view/filter/codec";
import { type Properties, type Property } from "@/platform/view/filter/types";
import { useConditions } from "@/platform/view/filter/useConditions";

interface PropertySubProps {
  itemKey: string;
  property: Omit<Property<unknown, any>, "decode" | "encode">;
  value: unknown;
  onChange: (value: unknown) => void;
}

const PropertySub = ({
  itemKey,
  property: { name, icon: PropertyIcon, Picker },
  value,
  onChange,
}: PropertySubProps): ReactElement => (
  <Base.Sub itemKey={itemKey}>
    <Base.SubTrigger>
      <PropertyIcon />
      {name}
    </Base.SubTrigger>
    <Picker value={value} onChange={onChange} />
  </Base.Sub>
);

export interface MenuProps<W> {
  properties: Properties<W>;
}

/**
 * The filter menu of a view: one item for each property, which opens the picker of the
 * property beside the menu. It is absent while the view is not editable.
 */
export const Menu = <W extends { and?: W[] }>({
  properties,
}: MenuProps<W>): ReactNode => {
  const { editable } = useContext("View.Filter.Menu");
  const [conditions, setConditions] = useConditions(properties);
  if (!editable) return null;
  return (
    <Dialog.Frame>
      <Dialog.Trigger
        hideCaret
        variant="text"
        size="medium"
        textColor={9}
        gap="small"
        tooltip="Filter"
        tooltipLocation={location.BOTTOM_RIGHT}
      >
        <Icon.Filter />
        Filter
      </Dialog.Trigger>
      <Base.Dialog aria-label="Filter">
        <Base.Menu gap="small">
          {Object.entries(properties).map(([key, property]) => (
            <PropertySub
              key={key}
              itemKey={key}
              property={property}
              value={find(conditions, key)}
              onChange={(value) =>
                setConditions(
                  value == null ? drop(conditions, key) : put(conditions, key, value),
                )
              }
            />
          ))}
        </Base.Menu>
      </Base.Dialog>
    </Dialog.Frame>
  );
};
