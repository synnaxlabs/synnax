export const applyCSSVars = (
  element: HTMLElement,
  vars: Record<string, string | number | undefined>
): void =>
  Object.entries(vars).forEach(
    ([key, value]) => value != null && element.style.setProperty(key, `${value}`)
  );
