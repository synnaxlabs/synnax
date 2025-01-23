import { type Layout } from "@/layout";
import { Configure, SEQUENCE_SELECTABLE } from "@/sequence/Configure";
import { SEQUENCE_TYPE } from "@/sequence/types";

export * from "@/sequence/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [SEQUENCE_TYPE]: Configure,
};

export const SELECTABLES: Layout.Selectable[] = [SEQUENCE_SELECTABLE];
