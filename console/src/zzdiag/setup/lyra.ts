import "zod/compile";

import { CSS } from "@synnaxlabs/lyra/css";
import { theme } from "@synnaxlabs/lyra/theme";
import { Theming } from "@synnaxlabs/lyra/theming";

CSS.applyVars(
  document.documentElement,
  Theming.toCSSVars(theme.themeZ.parse(theme.SYNNAX_LIGHT)),
);
