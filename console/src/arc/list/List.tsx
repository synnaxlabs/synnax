import { type arc } from "@synnaxlabs/client";
import { Flex, type Flux, Icon, Input, List as PList, Select } from "@synnaxlabs/pluto";
import { useState } from "react";

import { Item, type ItemProps } from "@/arc/list/Item";

export interface ListProps
  extends Pick<
      Flux.UseListReturn<PList.PagerParams, arc.Key, arc.Arc>,
      "data" | "getItem" | "subscribe" | "retrieve"
    >,
    Pick<ItemProps, "showLabels" | "showStatus"> {
  enableSearch?: boolean;
}

export const List = ({
  data,
  getItem,
  subscribe,
  retrieve,
  enableSearch = false,
  showLabels = true,
  showStatus = true,
}: ListProps) => {
  const { fetchMore, search } = PList.usePager({ retrieve });
  const [value, setValue] = useState<arc.Key[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  return (
    <Select.Frame<arc.Key, arc.Arc>
      multiple
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onChange={setValue}
      value={value}
      onFetchMore={fetchMore}
    >
      {enableSearch && (
        <Flex.Box x bordered style={{ padding: "2rem" }} background={1}>
          <Input.Text
            size="large"
            level="h4"
            variant="text"
            value={searchTerm}
            placeholder={
              <>
                {" "}
                <Icon.Search />
                Search Arcs...
              </>
            }
            onChange={(value) => {
              setSearchTerm(value);
              search(value);
            }}
          />
        </Flex.Box>
      )}
      <PList.Items<arc.Key>>
        {({ key, ...rest }) => (
          <Item key={key} {...rest} showLabels={showLabels} showStatus={showStatus} />
        )}
      </PList.Items>
    </Select.Frame>
  );
};
