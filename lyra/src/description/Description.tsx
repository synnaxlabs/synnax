// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/description/Description.css";

import { type text } from "@synnaxlabs/x";
import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { Text } from "@/text";

/** Where a {@link List} puts its values. */
export type Justify = "start" | "between";

/** Props for {@link List}. */
export interface ListProps extends ComponentPropsWithoutRef<"dl"> {
  /** Type scale step for every label and value. Defaults to "p". */
  level?: text.Level;
  /**
   * "start" puts each value beside its label. "between" pushes values to the end of the
   * list. Defaults to "start".
   */
  justify?: Justify;
}

const [Context, useContext] = context.create<text.Level>({
  defaultValue: "p",
  displayName: "Description.Context",
});

/**
 * A read-only list of labeled values. Labels share one column, so values line up.
 *
 * @example
 * <Description.List>
 *   <Description.Item>
 *     <Description.Label>Version</Description.Label>
 *     <Description.Value>{version}</Description.Value>
 *   </Description.Item>
 * </Description.List>
 */
export const List = ({
  className,
  level = "p",
  justify = "start",
  ...rest
}: ListProps): ReactElement => (
  <Context value={level}>
    <dl
      className={CSS.cls(
        CSS.B("description"),
        CSS.BM("description", justify),
        className,
      )}
      {...rest}
    />
  </Context>
);

/** Props for {@link Item}. */
export interface ItemProps extends ComponentPropsWithoutRef<"div"> {}

/** One label and its value. */
export const Item = ({ className, ...rest }: ItemProps): ReactElement => (
  <div className={CSS.cls(CSS.BE("description", "item"), className)} {...rest} />
);

/** Props for {@link Label}. */
export type LabelProps = Text.TextProps<"dt">;

/** The name of a value. */
export const Label = ({ className, color = 9, ...rest }: LabelProps): ReactElement => {
  const level = useContext();
  return (
    <Text.Text<"dt">
      el="dt"
      level={level}
      color={color}
      className={CSS.cls(CSS.BE("description", "label"), className)}
      {...rest}
    />
  );
};

/** Props for {@link Value}. */
export type ValueProps = Text.TextProps<"dd">;

/** A value. It wraps anywhere and can be selected for copying. */
export const Value = ({ className, color = 10, ...rest }: ValueProps): ReactElement => {
  const level = useContext();
  return (
    <Text.Text<"dd">
      el="dd"
      level={level}
      color={color}
      className={CSS.cls(CSS.BE("description", "value"), className)}
      {...rest}
    />
  );
};
