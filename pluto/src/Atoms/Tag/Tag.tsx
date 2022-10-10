import { AiOutlineClose } from "react-icons/ai";
import { ComponentSize } from "../../util/types";
import { IconText } from "../Typography";
import { ComponentSizeTypographyLevels, TextProps } from "../Typography";
import "./Tag.css";

export interface TagProps extends Omit<TextProps, "level"> {
  icon?: React.ReactElement;
  onClose?: () => void;
  color?: string;
  size?: ComponentSize;
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
  const closeIcon = onClose && (
    <AiOutlineClose
      className="pluto-tag__close"
      onClick={(e) => {
        e.stopPropagation();
        onClose();
      }}
    />
  );
  return (
    <IconText
      endIcon={closeIcon}
      startIcon={icon}
      className="pluto-tag"
      level={ComponentSizeTypographyLevels[size]}
      style={{
        border: `var(--pluto-border-width) solid ${color}`,
        backgroundColor: variant == "filled" ? color : "transparent",
        ...style,
      }}
      {...props}
    >
      {children}
    </IconText>
  );
}
