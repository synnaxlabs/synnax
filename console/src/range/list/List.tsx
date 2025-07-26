import { type ranger } from "@synnaxlabs/client";
import {
  Align,
  Component,
  type Flux,
  Icon,
  Input,
  List as PList,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { useState } from "react";

import { ListItem } from "@/range/list/Item";

const listItem = Component.renderProp(ListItem);

export interface ListProps
  extends Pick<
    Flux.UseListReturn<PList.PagerParams, ranger.Key, ranger.Range>,
    "data" | "getItem" | "subscribe" | "retrieve"
  > {
  enableSearch?: boolean;
}

export const List = ({
  data,
  getItem,
  subscribe,
  retrieve,
  enableSearch = false,
}: ListProps) => {
  const { fetchMore, search } = PList.usePager({ retrieve });
  const [value, setValue] = useState<ranger.Key[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  return (
    <Select.Frame<ranger.Key, ranger.Range>
      multiple
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onChange={setValue}
      value={value}
      onFetchMore={fetchMore}
    >
      {enableSearch && (
        <Align.Space x bordered style={{ padding: "2rem" }} background={1}>
          <Input.Text
            size="large"
            level="h4"
            variant="natural"
            value={searchTerm}
            placeholder={
              <Text.WithIcon level="h4" startIcon={<Icon.Search />}>
                Search Ranges...
              </Text.WithIcon>
            }
            onChange={(value) => {
              setSearchTerm(value);
              search(value);
            }}
          />
        </Align.Space>
      )}
      <PList.Items>{listItem}</PList.Items>
    </Select.Frame>
  );
};
