const classPrefix = " pluto-";

export const classList = (...classes: (string | undefined)[]): string => {
  return classes.join(" ");
};
