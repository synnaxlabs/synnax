import { TypographyLevel } from "../Atoms";
import { useThemeContext } from "./ThemeContext";

export const useFont = (level: TypographyLevel) => {
  const { theme } = useThemeContext();
  return `${theme.typography.p.weight} ${theme.typography.family} ${theme.typography.p.size}px`;
};
