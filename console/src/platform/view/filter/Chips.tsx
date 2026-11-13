// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type position } from "@synnaxlabs/lyra/position";
import { filter, location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useContext } from "@/platform/view/context";
import {
  Chip,
  ChipName,
  ChipRemove,
  ChipTrigger,
  ChipValue,
} from "@/platform/view/filter/Chip";
import { type Condition } from "@/platform/view/filter/codec";
import { type Properties, type Property } from "@/platform/view/filter/types";
import { useConditions } from "@/platform/view/filter/useConditions";

/** A picker opens below its chip. */
export const BELOW: position.LocationPreference = {
  targetCorner: location.BOTTOM_LEFT,
  dialogCorner: location.TOP_LEFT,
};

interface PropertyChipProps {
  property: Omit<Property<unknown, any>, "decode" | "encode">;
  value: unknown;
  onChange: (value: unknown) => void;
}

const PropertyChip = ({
  property: { name, icon, Operator, Value, Picker },
  value,
  onChange,
}: PropertyChipProps): ReactElement => (
  <Chip>
    <ChipName icon={icon}>{name}</ChipName>
    <Operator value={value} onChange={onChange} />
    <Dialog.Frame variant="floating" location={BELOW}>
      <ChipTrigger>
        <Value value={value} onChange={onChange} />
      </ChipTrigger>
      <Picker value={value} onChange={onChange} />
    </Dialog.Frame>
    <ChipRemove name={name} onClick={() => onChange(null)} />
  </Chip>
);

interface OtherChipProps {
  where: object;
  onRemove: () => void;
}

/** A node of the filter that no property reads. It shows as text. */
const OtherChip = ({ where, onRemove }: OtherChipProps): ReactElement => (
  <Chip>
    <ChipName icon={Icon.Filter}>Filter</ChipName>
    <ChipValue>{filter.describe(where)}</ChipValue>
    <ChipRemove name="Filter" onClick={onRemove} />
  </Chip>
);

export interface ChipsProps<W> {
  properties: Properties<W>;
}

/**
 * One chip for each condition of the view's filter, and a button that clears them all
 * when there are two or more. Without conditions, it is absent.
 */
export const Chips = <W extends { and?: W[] }>({
  properties,
}: ChipsProps<W>): ReactElement | null => {
  const { editable } = useContext("View.Filter.Chips");
  const [conditions, setConditions] = useConditions(properties);
  if (conditions.length === 0) return null;
  // A chip edits only its own condition, as a filter can hold two on one property.
  const replace = (index: number, next: Condition<W> | null): void =>
    setConditions(
      next == null
        ? conditions.filter((_, i) => i !== index)
        : conditions.map((c, i) => (i === index ? next : c)),
    );
  return (
    <>
      {conditions.map((condition, i) =>
        condition.property == null ? (
          <OtherChip
            key={`other-${i}`}
            where={condition.where}
            onRemove={() => replace(i, null)}
          />
        ) : (
          <PropertyChip
            key={`${condition.property}-${i}`}
            property={properties[condition.property]}
            value={condition.value}
            onChange={(value) =>
              replace(i, value == null ? null : { property: condition.property, value })
            }
          />
        ),
      )}
      {editable && conditions.length > 1 && (
        <Button.Button
          variant="text"
          size="small"
          textColor={9}
          onClick={() => setConditions([])}
        >
          Clear
        </Button.Button>
      )}
    </>
  );
};
