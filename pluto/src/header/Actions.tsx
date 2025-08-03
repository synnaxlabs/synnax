// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array } from "@synnaxlabs/x";
import { Fragment, isValidElement, type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { useContext } from "@/header/Header";
import { Text } from "@/text";

export type ActionSpec = Button.ButtonProps | ReactElement;

export interface ActionsProps extends Omit<Flex.BoxProps, "children" | "direction"> {
  children?: ActionSpec | ActionSpec[];
}

/**
 * Custom actions to render on the right side of the header.
 *
 * @param children - The actions to render. If the action is of type
 * {@link ButtonIconProps}, a correectly sized {@link ButtonIconOnly} is rendered
 * using the given props. If the action is a JSX element, it is renderered directly.
 * It's a good idea to prefer the latter in almost all cases for simplicity.
 */
export const Actions = ({ children = [], ...rest }: ActionsProps): ReactElement => {
  const { level, divided } = useContext();
  return (
    <Flex.Box
      x
      gap="small"
      align="center"
      className={CSS.BE("header", "actions")}
      {...rest}
    >
      {array.toArray(children).map((action, i) => (
        <Action key={i} index={i} level={level} divided={divided}>
          {action}
        </Action>
      ))}
    </Flex.Box>
  );
};

interface ActionProps {
  index: number;
  level: Text.Level;
  children: ReactElement | Button.ButtonProps;
  divided: boolean;
}

const Action = ({ index, level, children, divided }: ActionProps): ReactElement => {
  let content: ReactElement = children as ReactElement;
  if (!isValidElement(children)) {
    const { onClick, key, ...rest } = children;
    content = (
      <Button.Button
        key={key ?? index}
        onClick={(e) => {
          e.stopPropagation();
          e.preventDefault();
          onClick?.(e);
        }}
        size={Text.LEVEL_COMPONENT_SIZES[level]}
        {...rest}
      />
    );
  }
  return (
    <Fragment key={index}>
      {divided && <Divider.Divider />}
      {content}
    </Fragment>
  );
};
