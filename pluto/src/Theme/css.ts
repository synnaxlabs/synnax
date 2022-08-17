import { Theme } from "./theme";

export const applyThemeAsCssVars = (element: HTMLElement, theme: Theme) => {
  // |||| COLORS ||||

  // || PRIMARY ||

  setProperty(element, "--primary-m1", theme.colors.primary.m1);
  setProperty(element, "--primary-z", theme.colors.primary.z);
  setProperty(element, "--primary-p1", theme.colors.primary.p1);

  // || GRAY ||

  setProperty(element, "--gray-m2", theme.colors.gray.m2);
  setProperty(element, "--gray-m1", theme.colors.gray.m1);
  setProperty(element, "--gray-z", theme.colors.gray.z);
  setProperty(element, "--gray-p1", theme.colors.gray.p1);
  setProperty(element, "--gray-p2", theme.colors.gray.p2);

  // || ERROR ||

  setProperty(element, "--error-m1", theme.colors.error.m1);
  setProperty(element, "--error-z", theme.colors.error.z);
  setProperty(element, "--error-p1", theme.colors.error.p1);

  // || WHITE, BLACK, BACKGROUND, TEXT ||

  setProperty(element, "--white", theme.colors.white);
  setProperty(element, "--black", theme.colors.black);
  setProperty(element, "--background", theme.colors.background);
  setProperty(element, "--text-color", theme.colors.text);

  // |||| SIZES ||||

  setProperty(element, "--base-size", theme.sizes.base);
  setProperty(element, "--border-radius", theme.sizes.border.radius);
  setProperty(element, "--border-width", theme.sizes.border.width);
  setProperty(element, "--font-family", theme.typography.family);

  // |||| TYPOGRAPHY ||||

  // || H1 ||

  setProperty(element, "--h1-font-size", theme.typography.h1.size);
  setProperty(element, "--h1-line-height", theme.typography.h1.lineHeight);
  setProperty(element, "--h1-weight", theme.typography.h1.weight);

  // || H2 ||

  setProperty(element, "--h2-font-size", theme.typography.h2.size);
  setProperty(element, "--h2-line-height", theme.typography.h2.lineHeight);
  setProperty(element, "--h2-weight", theme.typography.h2.weight);

  // || H3 ||

  setProperty(element, "--h3-font-size", theme.typography.h3.size);
  setProperty(element, "--h3-line-height", theme.typography.h3.lineHeight);
  setProperty(element, "--h3-weight", theme.typography.h3.weight);

  // || H4 ||

  setProperty(element, "--h4-font-size", theme.typography.h4.size);
  setProperty(element, "--h4-line-height", theme.typography.h4.lineHeight);
  setProperty(element, "--h4-weight", theme.typography.h4.weight);

  // || H5 ||

  setProperty(element, "--h5-font-size", theme.typography.h5.size);
  setProperty(element, "--h5-line-height", theme.typography.h5.lineHeight);
  setProperty(element, "--h5-weight", theme.typography.h5.weight);
  setProperty(
    element,
    "--h5-text-transform",
    theme.typography.h2.textTransform
  );

  // || P ||

  setProperty(element, "--p-font-size", theme.typography.p.size);
  setProperty(element, "--p-line-height", theme.typography.p.lineHeight);
  setProperty(element, "--p-weight", theme.typography.p.weight);

  // || SMALL ||

  setProperty(element, "--small-font-size", theme.typography.small.size);
  setProperty(
    element,
    "--small-line-height",
    theme.typography.small.lineHeight
  );
  setProperty(element, "--small-weight", theme.typography.small.weight);
};

const setProperty = (
  element: HTMLElement,
  property: string,
  value: string | number | null | undefined
) => {
  if (value === null || value === undefined) {
    return;
  }
  if (typeof value === "number") {
    value = value + "px";
  }
  element.style.setProperty(property, String(value));
};
