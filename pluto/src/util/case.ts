export const camelToTitle = (str: string): string => {
  return str.replace(/([A-Z])/g, " $1").replace(/^./, (str) => str.toUpperCase());
};
