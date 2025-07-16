import { type Component, type Flux, type List, type state } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

export interface RetrieveParams extends Flux.Params {
  term?: string;
  offset?: number;
  limit?: number;
}

export interface UseListReturn<E extends record.Keyed<string>>
  extends Pick<List.FrameProps<string, E>, "getItem" | "subscribe"> {
  data: string[];
  handleSelect: (key: string) => void;
  listItem: Component.RenderProp<List.ItemProps<string>>;
  retrieve: (params: state.SetArg<RetrieveParams>) => void;
}
