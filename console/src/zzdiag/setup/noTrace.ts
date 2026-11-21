import "zod/compile";
import "@synnaxlabs/lyra/dist/base.css";
import "@synnaxlabs/pluto/dist/pluto.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { setupBrowser } from "@synnaxlabs/lyra/styletest";
import { theme } from "@synnaxlabs/lyra/theme";
import { Theming } from "@synnaxlabs/lyra/theming";
import { failOnConsoleOutput } from "@synnaxlabs/x/testutil";

failOnConsoleOutput({
  ignore: [/^Error: ResizeObserver loop completed with undelivered notifications\.$/],
});
setupBrowser();
CSS.applyVars(
  document.documentElement,
  Theming.toCSSVars(theme.themeZ.parse(theme.SYNNAX_LIGHT)),
);
const { configure } = await import("@testing-library/react");
configure({ asyncUtilTimeout: 5000 });
const { preloadTimeLanguage } = await import("@synnaxlabs/lyra/testutil");
preloadTimeLanguage();
