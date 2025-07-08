import { type Component, type Flux, type List } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

export interface RetrieveParams extends Flux.Params {
  term?: string;
  offset?: number;
  limit?: number;
}

export interface UseListReturn<E extends record.Keyed<string>> {
  data: string[];
  useListItem: (key?: string) => E | undefined;
  handleSelect: (key: string) => void;
  listItem: Component.RenderProp<List.ItemProps<string>>;
  retrieve: (params: RetrieveParams) => void;
}
