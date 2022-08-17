import { ThemeProvider } from "../src/Theme/ThemeContext.tsx";
import {aryaDark, aryaLight} from "../src/Theme/theme.ts";

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
  (Story) => <ThemeProvider themes={[aryaDark]}>{Story()}</ThemeProvider>,
];
