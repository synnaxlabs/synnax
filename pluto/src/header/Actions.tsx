// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { toArray } from "@synnaxlabs/x";
import { Fragment, isValidElement, type ReactElement } from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { useContext } from "@/header/Header";
import { Text } from "@/text";

export type ActionSpec = Button.IconProps | ReactElement;

export interface ActionsProps {
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
export const Actions = ({ children = [] }: ActionsProps): ReactElement => {
  const { level, divided } = useContext();
  return (
    <Align.Space
      direction="x"
      size="small"
      align="center"
      className={CSS.BE("header", "actions")}
    >
      {toArray(children).map((action, i) => (
        <Action key={i} index={i} level={level} divided={divided}>
          {action}
        </Action>
      ))}
    </Align.Space>
  );
};

interface ActionProps {
  index: number;
  level: Text.Level;
  children: ReactElement | Button.IconProps;
  divided: boolean;
}

const Action = ({ index, level, children, divided }: ActionProps): ReactElement => {
  let content: ReactElement = children as ReactElement;
  if (!isValidElement(children)) {
    const { onClick, ...props } = children as Button.IconProps;
    content = (
      <Button.Icon
        onClick={(e) => {
          e.stopPropagation();
          e.preventDefault();
          onClick?.(e);
        }}
        key={index}
        size={Text.LevelComponentSizes[level]}
        {...props}
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
