import { cloneElement, Fragment, isValidElement, ReactElement } from "react";

import { Button, ButtonIconOnlyProps } from "../Button";
import { Divider } from "../Divider";
import { Space } from "../Space";
import { Typography, TypographyLevel } from "../Typography";

import { useHeaderContext } from "./Header";

export type HeaderAction = ButtonIconOnlyProps | ReactElement;

interface HeaderActionsProps {
  children: Array<ButtonIconOnlyProps | ReactElement>;
}

export const HeaderActions = ({ children }: HeaderActionsProps): JSX.Element => {
  const { level, divided } = useHeaderContext();
  return (
    <Space
      direction="horizontal"
      size="small"
      align="center"
      className="pluto-header__actions"
    >
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
