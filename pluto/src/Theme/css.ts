import { Theme } from "./theme";

export const applyThemeAsCssVars = (element: HTMLElement, theme: Theme) => {
  // |||| COLORS ||||

  // || PRIMARY ||

  setProperty(element, "--pluto-primary-m1", theme.colors.primary.m1);
  setProperty(element, "--pluto-primary-z", theme.colors.primary.z);
  setProperty(element, "--pluto-primary-p1", theme.colors.primary.p1);
  setOpacityVariants(element, "--pluto-primary-p1", theme.colors.primary.z);

  // || GRAY ||

  setProperty(element, "--pluto-gray-m3", theme.colors.gray.m3);
  setProperty(element, "--pluto-gray-m2", theme.colors.gray.m2);
  setProperty(element, "--pluto-gray-m1", theme.colors.gray.m1);
  setProperty(element, "--pluto-gray-m0", theme.colors.gray.m0);
  setOpacityVariants(element, "--pluto-gray-m0", theme.colors.gray.m0);
  setOpacityVariants(element, "--pluto-gray-p0", theme.colors.gray.m0);
  setProperty(element, "--pluto-gray-p0", theme.colors.gray.p0);
  setProperty(element, "--pluto-gray-p1", theme.colors.gray.p1);
  setProperty(element, "--pluto-gray-p2", theme.colors.gray.p2);
  setProperty(element, "--pluto-gray-p3", theme.colors.gray.p3);

  // || ERROR ||

  setProperty(element, "--pluto-error-m1", theme.colors.error.m1);
  setProperty(element, "--pluto-error-z", theme.colors.error.z);
  setProperty(element, "--pluto-error-p1", theme.colors.error.p1);

  // || WHITE, BLACK, BACKGROUND, TEXT ||

  setProperty(element, "--pluto-white", theme.colors.white);
  setProperty(element, "--pluto-black", theme.colors.black);
  setProperty(element, "--pluto-background-color", theme.colors.background);
  setProperty(element, "--pluto-text-color", theme.colors.text);
  setProperty(element, "--pluto-border-color", theme.colors.border);

  // |||| SIZES ||||

  setProperty(element, "--pluto-base-size", theme.sizes.base);
  setProperty(element, "--pluto-border-radius", theme.sizes.border.radius);
  setProperty(element, "--pluto-border-width", theme.sizes.border.width);
  setProperty(element, "--pluto-font-family", theme.typography.family);

  // |||| TYPOGRAPHY ||||

  // || H1 ||

  setProperty(element, "--pluto-h1-size", theme.typography.h1.size);
  setProperty(
    element,
    "--pluto-h1-line-height",
    theme.typography.h1.lineHeight
  );
  setProperty(element, "--pluto-h1-weight", theme.typography.h1.weight);

  // || H2 ||

  setProperty(element, "--pluto-h2-size", theme.typography.h2.size);
  setProperty(
    element,
    "--pluto-h2-line-height",
    theme.typography.h2.lineHeight
  );
  setProperty(element, "--pluto-h2-weight", theme.typography.h2.weight);

  // || H3 ||

  setProperty(element, "--pluto-h3-size", theme.typography.h3.size);
  setProperty(
    element,
    "--pluto-h3-line-height",
    theme.typography.h3.lineHeight
  );
  setProperty(element, "--pluto-h3-weight", theme.typography.h3.weight);

  // || H4 ||

  setProperty(element, "--pluto-h4-size", theme.typography.h4.size);
  setProperty(
    element,
    "--pluto-h4-line-height",
    theme.typography.h4.lineHeight
  );
  setProperty(element, "--pluto-h4-weight", theme.typography.h4.weight);

  // || H5 ||

  setProperty(element, "--pluto-h5-size", theme.typography.h5.size);
  setProperty(
    element,
    "--pluto-h5-line-height",
    theme.typography.h5.lineHeight
  );
  setProperty(element, "--pluto-h5-weight", theme.typography.h5.weight);
  setProperty(
    element,
    "--pluto-h5-text-transform",
    theme.typography.h2.textTransform
  );

  // || P ||

  setProperty(element, "--pluto-p-size", theme.typography.p.size);
  setProperty(element, "--pluto-p-line-height", theme.typography.p.lineHeight);
  setProperty(element, "--pluto-p-weight", theme.typography.p.weight);

  // || SMALL ||

  setProperty(element, "--pluto-small-size", theme.typography.small.size);
  setProperty(
    element,
    "--pluto-small-line-height",
    theme.typography.small.lineHeight
  );
  setProperty(element, "--pluto-small-weight", theme.typography.small.weight);
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

const opacityVariants = {
  "80": "CC",
  "60": "99",
  "40": "66",
  "20": "33",
};

const setOpacityVariants = (
  element: HTMLElement,
  baseVar: string,
  color: string | number
) => {
  Object.entries(opacityVariants).forEach(([key, opacity]) => {
    setProperty(element, `${baseVar}-${key}`, `${color}${opacity}`);
  });
};
