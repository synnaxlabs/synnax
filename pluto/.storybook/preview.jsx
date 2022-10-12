import { ThemeProvider } from "../src/Theme/ThemeContext.tsx";
import {synnaxDark, synnaxLight} from "../src/Theme/theme.ts";
import "./index.css"

export const parameters = {
  actions: { argTypesRegex: "^on[A-Z].*" },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
};

export const decorators = [
  (Story) => <ThemeProvider themes={[synnaxDark]}>{Story()}</ThemeProvider>,
];
