import clsx from "clsx";
import { Fragment, ReactElement, cloneElement, isValidElement } from "react";
import { Button, ButtonIconOnlyProps } from "@/atoms/Button";
import { Divider } from "@/atoms/Divider";
import { Space, SpaceProps } from "@/atoms/Space";
import {
  Text,
  TextProps,
  TypographyLevelComponentSizes,
} from "@/atoms/Typography";
import "./Header.css";

export interface HeaderProps extends Omit<SpaceProps, "children">, TextProps {
  divided?: boolean;
  icon?: ReactElement;
  actions?: (ButtonIconOnlyProps | ReactElement)[];
}

export const Header = ({
  icon,
  level = "h1",
  divided = false,
  children,
  className,
  actions,
  ...props
}: HeaderProps) => {
  return (
    <Space
      direction="horizontal"
      justify="spaceBetween"
      className={clsx(`pluto-header pluto-bordered--bottom`, className)}
      empty
      {...props}
    >
      <Text.WithIcon
        level={level}
        startIcon={icon}
        divided={divided}
        className="pluto-header__text"
      >
        {children}
      </Text.WithIcon>
      <Space
        direction="horizontal"
        size="small"
        align="center"
        className="pluto-header__actions"
      >
        {actions &&
          actions.map((action, i) => {
            const content = isValidElement(action) ? (
              cloneElement(action, { key: action.key })
            ) : (
              <Button.IconOnly
                onClick={action.onClick}
                key={i}
                size={TypographyLevelComponentSizes[level]}
                {...action}
              >
                {action.children}
              </Button.IconOnly>
            );
            return (
              <Fragment key={i}>
                {divided && <Divider direction="vertical" />}
                {content}
              </Fragment>
            );
          })}
      </Space>
    </Space>
  );
};
