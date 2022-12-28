export type KeyedRecord<E extends Record<string, unknown> = Record<string, unknown>> = {
  key: string;
} & Partial<Record<keyof E, unknown>>;

type RenderableValue = string | number | undefined;

export type UnknownRecord<E extends Record<string, unknown> = Record<string, unknown>> =
  Record<keyof E, unknown>;

export type RenderableRecord<
  E extends Record<string, RenderableValue> = Record<string, RenderableValue>
> = {
  key: string;
} & Partial<Record<keyof E, RenderableValue>>;
