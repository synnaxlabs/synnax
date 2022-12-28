interface Stringer {
  toString: () => string | number | undefined;
}

export type PureRenderableValue = string | number | undefined;
export type RenderableValue = PureRenderableValue | Stringer;

export const render = (value: RenderableValue): string | number | undefined => {
  if (value === undefined || typeof value === "string" || typeof value === "number")
    return value;
  if (value.toString === undefined) throw new Error("invalid renderer");
  return value.toString();
};
