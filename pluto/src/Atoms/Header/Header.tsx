import Space, { SpaceProps } from "../Space/Space";
import "./Header.css";
import { ReactElement } from "react";
import clsx from "clsx";
import IconText from "../Typography/IconText";
import Button from "../Button/Button";
import { TypographyLevel } from "../Typography";

export interface HeadingProps extends Omit<SpaceProps, "children" | "size"> {
  level: TypographyLevel;
  text: string;
  icon?: ReactElement;
  actions?: Action[];
}

type Action = {
  icon: ReactElement;
  onClick: () => void;
};

const Header = ({
  level,
  text,
  icon,
  style,
  className,
  actions = [],
  ...props
}: HeadingProps) => {
  return (
    <Space
      direction="horizontal"
      className={clsx("pluto-header__container", className)}
      align="center"
      style={style}
      justify="spaceBetween"
      {...props}
    >
      <IconText level={level} startIcon={icon}>
        {text}
      </IconText>
      <Space direction="horizontal">
        {actions?.map(({ icon }) => {
          return <Button.IconOnly>{icon}</Button.IconOnly>;
        })}
      </Space>
    </Space>
  );
};

export default Header;
