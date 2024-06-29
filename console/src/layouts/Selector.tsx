import { v4 as uuidv4 } from "uuid";

import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Schematic } from "@/schematic";

const SELECTABLES: Layout.Selectable[] = [LinePlot.SELECTABLE, Schematic.SELECTABLE];

export const SELECTOR_TYPE = "visLayoutSelector";

export const createSelector = (
  props: Omit<Partial<Layout.State>, "type">,
): Omit<Layout.State, "windowKey"> => {
  const {
    location = "mosaic",
    name = "New Layout",
    key = uuidv4(),
    window,
    tab,
  } = props;
  return {
    type: SELECTOR_TYPE,
    location,
    name,
    key,
    window,
    tab,
  };
};

export const Selector = Layout.createSelectorComponent(SELECTABLES);
