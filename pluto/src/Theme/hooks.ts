import { TypographyLevel } from "../Atoms";
import { useThemeContext } from "./ThemeContext";

export const useFont = (level: TypographyLevel) => {
  const {
    theme: { typography },
  } = useThemeContext();
  const { weight, size } = typography[level];
  return `${weight} ${typography.family} ${size}rem`;
};
