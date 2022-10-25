import { AiOutlineClose } from "react-icons/ai";
import { ComponentSize } from "@/util";
import {
  Text,
  ComponentSizeTypographyLevels,
  TextProps,
} from "@/atoms/Typography";
import "./Tag.css";

export interface TagProps extends Omit<TextProps, "level"> {
  icon?: React.ReactElement;
  onClose?: () => void;
  color?: string;
  size?: ComponentSize;
  variant?: "filled" | "outlined";
}

export const Tag = ({
  children = "",
  size = "medium",
  variant = "filled",
  color = "var(--pluto-primary-z)",
  icon,
  onClose,
  style,
  ...props
}: TagProps) => {
  const closeIcon = onClose && (
    <AiOutlineClose
      aria-label="close"
      className="pluto-tag__close"
      onClick={(e) => {
        e.stopPropagation();
        onClose();
      }}
    />
  );
  return (
    <Text.WithIcon
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
    </Text.WithIcon>
  );
};
