// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/view/filter/Chip.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useContext } from "@/platform/view/context";

const SEGMENT_CLASS = CSS.BE("view", "filter-segment");

export interface ChipProps extends Flex.BoxProps {}

/**
 * One condition of a view's filter, as parts split by lines: a {@link ChipName}, a
 * {@link ChipOperator}, a {@link ChipValue} or {@link ChipTrigger}, and a
 * {@link ChipRemove}.
 */
export const Chip = ({ className, ...rest }: ChipProps): ReactElement => (
  <Flex.Box
    x
    align="stretch"
    empty
    bordered
    borderColor={5}
    rounded="medium"
    background={2}
    className={CSS.cls(CSS.BE("view", "filter-chip"), className)}
    {...rest}
  />
);

export interface ChipNameProps extends PropsWithChildren {
  /** The icon of the property the condition tests. */
  icon: Icon.FC;
}

/** The name of the property a {@link Chip} tests, after its icon. */
export const ChipName = ({
  icon: PropertyIcon,
  children,
}: ChipNameProps): ReactElement => (
  <Text.Text gap="small" align="center" className={SEGMENT_CLASS}>
    <PropertyIcon color={9} />
    {children}
  </Text.Text>
);

export interface ChipOperatorProps extends PropsWithChildren {}

/** How a {@link Chip} compares the property to its value, such as "is". */
export const ChipOperator = ({ children }: ChipOperatorProps): ReactElement => (
  <Text.Text color={9} align="center" className={SEGMENT_CLASS}>
    {children}
  </Text.Text>
);

export interface ChipValueProps extends PropsWithChildren {}

/** The value a {@link Chip} compares the property to. */
export const ChipValue = ({ children }: ChipValueProps): ReactElement => (
  <Text.Text el="div" gap="small" align="center" className={SEGMENT_CLASS}>
    {children}
  </Text.Text>
);

export interface ChipTriggerProps extends Omit<
  Dialog.TriggerProps,
  "variant" | "size" | "level" | "hideCaret" | "sharp" | "preview"
> {}

/**
 * A part of a {@link Chip} that opens a dialog, such as the picker of its value or a
 * choice of its operator. It opens nothing while the view is not editable.
 */
export const ChipTrigger = ({ className, ...rest }: ChipTriggerProps): ReactElement => {
  const { editable } = useContext("View.Filter.ChipTrigger");
  return (
    <Dialog.Trigger
      variant="text"
      size="small"
      level="p"
      gap="small"
      hideCaret
      sharp
      preview={!editable}
      className={CSS.cls(SEGMENT_CLASS, CSS.BE("view", "filter-trigger"), className)}
      {...rest}
    />
  );
};

export interface ChipRemoveProps {
  /** The name of the property, which names the button for assistive technology. */
  name: string;
  onClick: () => void;
}

/**
 * Removes the condition of a {@link Chip} from the filter. It is absent while the view
 * is not editable.
 */
export const ChipRemove = ({ name, onClick }: ChipRemoveProps): ReactElement | null => {
  const { editable } = useContext("View.Filter.ChipRemove");
  if (!editable) return null;
  return (
    <Button.Button
      variant="text"
      size="small"
      square
      sharp
      textColor={8}
      className={CSS.BE("view", "filter-remove")}
      aria-label={`Remove ${name} filter`}
      tooltip="Remove filter"
      tooltipLocation="bottom"
      onClick={onClick}
    >
      <Icon.Close />
    </Button.Button>
  );
};
