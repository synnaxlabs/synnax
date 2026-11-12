// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/view/FilterChip.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useContext } from "@/platform/view/context";

const SEGMENT_CLASS = CSS.BE("view", "filter-segment");

export interface FilterChipProps extends Flex.BoxProps {}

/**
 * One condition of a view's filter, as parts split by lines: a {@link FilterChipName},
 * a {@link FilterChipOperator}, a {@link FilterChipValue} or {@link FilterChipTrigger},
 * and a {@link FilterChipRemove}.
 */
export const FilterChip = ({ className, ...rest }: FilterChipProps): ReactElement => (
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

export interface FilterChipNameProps extends PropsWithChildren {
  /** The icon of the property the condition tests. */
  icon: Icon.FC;
}

/** The name of the property a {@link FilterChip} tests, after its icon. */
export const FilterChipName = ({
  icon: PropertyIcon,
  children,
}: FilterChipNameProps): ReactElement => (
  <Text.Text gap="small" align="center" className={SEGMENT_CLASS}>
    <PropertyIcon color={9} />
    {children}
  </Text.Text>
);

export interface FilterChipOperatorProps extends PropsWithChildren {}

/** How a {@link FilterChip} compares the property to its value, such as "is". */
export const FilterChipOperator = ({
  children,
}: FilterChipOperatorProps): ReactElement => (
  <Text.Text color={9} align="center" className={SEGMENT_CLASS}>
    {children}
  </Text.Text>
);

export interface FilterChipValueProps extends PropsWithChildren {}

/** The value a {@link FilterChip} compares the property to. */
export const FilterChipValue = ({ children }: FilterChipValueProps): ReactElement => (
  <Text.Text el="div" gap="small" align="center" className={SEGMENT_CLASS}>
    {children}
  </Text.Text>
);

export interface FilterChipTriggerProps extends Omit<
  Dialog.TriggerProps,
  "variant" | "size" | "level" | "hideCaret" | "sharp" | "preview"
> {}

/**
 * A part of a {@link FilterChip} that opens a dialog, such as a picker of its value or
 * a choice of its operator. It opens nothing while the view is not editable.
 */
export const FilterChipTrigger = ({
  className,
  ...rest
}: FilterChipTriggerProps): ReactElement => {
  const { editable } = useContext("View.FilterChipTrigger");
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

export interface FilterChipRemoveProps {
  /** The name of the property, which names the button for assistive technology. */
  name: string;
  onClick: () => void;
}

/** Removes the condition of a {@link FilterChip} from the filter. */
export const FilterChipRemove = ({
  name,
  onClick,
}: FilterChipRemoveProps): ReactElement => (
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
