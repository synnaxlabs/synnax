// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cloneElement, Fragment, isValidElement, ReactElement } from "react";

import { useHeaderContext } from "./Header";

import { Button, ButtonIconOnlyProps } from "@/core/Button";
import { Divider } from "@/core/Divider";
import { Space } from "@/core/Space";
import { Typography, TypographyLevel } from "@/core/Typography";

export type HeaderAction = ButtonIconOnlyProps | ReactElement;

export interface HeaderActionsProps {
  children: Array<ButtonIconOnlyProps | ReactElement>;
}

export const HeaderActions = ({ children }: HeaderActionsProps): JSX.Element => {
  const { level, divided } = useHeaderContext();
  return (
    <Space direction="x" size="small" align="center" className="pluto-header__actions">
      {children?.map((action, i) => (
        <HeaderActionC key={i} index={i} level={level} divided={divided}>
          {action}
        </HeaderActionC>
      ))}
    </Space>
  );
};

interface HeaderActionCProps {
  index: number;
  level: TypographyLevel;
  children: ReactElement | ButtonIconOnlyProps;
  divided: boolean;
}

const HeaderActionC = ({
  index,
  level,
  children,
  divided,
}: HeaderActionCProps): JSX.Element => {
  const content = isValidElement(children) ? (
    cloneElement(children, { key: children.key })
  ) : (
    <Button.IconOnly
      onClick={(e) => {
        e.stopPropagation();
        e.preventDefault();
        children.onClick?.(e);
      }}
      key={index}
      size={Typography.LevelComponentSizes[level]}
      {...children}
    >
      {children.children}
    </Button.IconOnly>
  );
  return (
    <Fragment key={index}>
      {divided && <Divider />}
      {content}
    </Fragment>
  );
};
