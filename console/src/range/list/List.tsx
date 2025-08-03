import { type ranger } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  type Flux,
  Icon,
  Input,
  List as PList,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { useState } from "react";

import { Item, type ItemProps } from "@/range/list/Item";

export interface ListProps
  extends Pick<
      Flux.UseListReturn<PList.PagerParams, ranger.Key, ranger.Range>,
      "data" | "getItem" | "subscribe" | "retrieve"
    >,
    Pick<ItemProps, "showParent" | "showLabels" | "showTimeRange" | "showFavorite"> {
  enableSearch?: boolean;
}

export const List = ({
  data,
  getItem,
  subscribe,
  retrieve,
  enableSearch = false,
  showParent = true,
  showLabels = true,
  showTimeRange = true,
  showFavorite = true,
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
        <Flex.Box
          x
          bordered
          style={{ padding: "2rem" }}
          background={1}
          justify="between"
        >
          <Input.Text
            size="small"
            level="h4"
            variant="text"
            value={searchTerm}
            placeholder={
              <Flex.Box x align="center" gap="tiny">
                <Icon.Search />
                <Text.Text level="h4">Search Ranges...</Text.Text>
              </Flex.Box>
            }
            onChange={(value) => {
              setSearchTerm(value);
              search(value);
            }}
          />
          <Button.Button>
            <Icon.Filter />
          </Button.Button>
        </Flex.Box>
      )}
      <PList.Items<string>>
        {({ key, ...rest }) => (
          <Item
            key={key}
            {...rest}
            showParent={showParent}
            showLabels={showLabels}
            showTimeRange={showTimeRange}
            showFavorite={showFavorite}
          />
        )}
      </PList.Items>
    </Select.Frame>
  );
};
