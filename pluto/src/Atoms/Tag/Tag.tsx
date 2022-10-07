import { CSSProperties } from "react";
import { AiOutlineClose } from "react-icons/ai";
import IconText, { IconTextProps } from "../Typography/IconText";
import { Size, sizeLevels, TextProps } from "../Typography/Text";
import "./Tag.css";

export interface TagProps extends Omit<TextProps, "level"> {
  icon?: React.ReactElement;
  onClose?: () => void;
  color?: string | number;
  size?: Size;
  variant?: "filled" | "outlined";
}

export default function Tag({
  children = "",
  size = "medium",
  variant = "filled",
  color = "var(--pluto-primary-z)",
  icon,
  onClose,
  style,
  ...props
}: TagProps) {
  let closeIcon = undefined;
  if (onClose) {
    closeIcon = (
      <AiOutlineClose
        className="pluto-tag__close"
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
      />
    );
  }

  const _style: CSSProperties = {
    border: `var(--pluto-border-width) solid ${color}`,
  };
  if (variant == "filled") {
    _style.backgroundColor = color;
  }

  return (
    <IconText
      endIcon={closeIcon}
      startIcon={icon}
      className="pluto-tag"
      level={sizeLevels[size]}
      style={{ ..._style, ...style }}
      {...props}
    >
      {children}
    </IconText>
  );
}
