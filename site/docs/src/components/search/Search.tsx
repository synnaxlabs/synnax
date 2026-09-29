// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb } from "@synnaxlabs/lyra/breadcrumb";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { caseconv, deep } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef, useState } from "react";
import z from "zod";

import { codifyHTML } from "@/components/text/InlineCode";

interface SearchResult {
  key: string;
  title: string;
  description: string;
  content: string;
  href: string;
}
const ALGOLIA_APP_ID = "YWD9T0JXCS";
const ALGOLIA_SEARCH_ONLY_API_KEY = "1f8b0497301392c94adedf89a98afb6f";
const ALGOLIA_URL = `https://${ALGOLIA_APP_ID}-dsn.algolia.net/1/indexes/docs_site/query`;
const ALGOLIA_HEADERS = {
  "X-Algolia-API-Key": ALGOLIA_SEARCH_ONLY_API_KEY,
  "X-Algolia-Application-Id": ALGOLIA_APP_ID,
};

export const Search = (): ReactElement => (
  <Triggers.Provider>
    <Dialog.Frame variant="modal" className="search-box">
      <Dialog.Trigger
        variant="outlined"
        justify="center"
        size="medium"
        textColor={8}
        trigger={["Control", "K"]}
        triggerIndicator
      >
        <Icon.Search />
        Search
      </Dialog.Trigger>
      <Dialog.Dialog
        bordered={false}
        pack
        rounded={1}
        className="search-results__content"
      >
        <SearchDialogContent />
      </Dialog.Dialog>
    </Dialog.Frame>
  </Triggers.Provider>
);

// The first key found in a result's href picks its icon, so specific keys come first.
const ICONS: Record<string, Icon.ReactElement> = {
  "opc-ua": <Icon.Logo.OPCUA />,
  "/ni/": <Icon.Logo.NI />,
  "get-started": <Icon.Bolt />,
  channels: <Icon.Channel />,
  hardware: <Icon.Device />,
  visualize: <Icon.Visualize />,
  ranges: <Icon.Range />,
  control: <Icon.Control />,
  alarms: <Icon.Status />,
  client: <Icon.Terminal />,
  enterprise: <Icon.Core />,
  blog: <Icon.Guide />,
  releases: <Icon.Release />,
};

export const SearchListItem = (props: List.ItemRenderProps<string>) => {
  const { itemKey } = props;
  const item = List.useItem<string, SearchResult>(itemKey);
  if (item == null) return null;
  const { href, title, content } = item;
  const icon = Object.entries(ICONS).find(([k]) => href.includes(k))?.[1];
  const path = deep.transformPath(
    href,
    (part, index, parts) => {
      if (part.length === 0 || index === parts.length - 1) return undefined;
      const split = part
        .split("-")
        .filter((p) => p.length > 0)
        .map(caseconv.capitalize);
      return split.join(" ");
    },
    "/",
  );
  return (
    <Select.Item<string, "a">
      id={itemKey}
      el="a"
      direction="y"
      style={{ padding: "2.5rem 3rem" }}
      gap="medium"
      aria-selected
      href={href}
      propagateClick
      {...props}
    >
      <Flex.Box direction="y" empty>
        <Text.Text
          level="h5"
          dangerouslySetInnerHTML={{ __html: codifyHTML(title) }}
          empty
        />
        {path.length > 0 && (
          <Breadcrumb.Breadcrumb level="small" gap="tiny" highlightVariant="last">
            {icon}
            {path.split("/").map((segment, index) => (
              <Breadcrumb.Segment key={index} color={8}>
                {segment}
              </Breadcrumb.Segment>
            ))}
          </Breadcrumb.Breadcrumb>
        )}
      </Flex.Box>
      <Text.Text level="small" dangerouslySetInnerHTML={{ __html: content }} />
    </Select.Item>
  );
};

const searchListItem = Component.renderProp(SearchListItem);

const hitSchema = z.object({
  objectID: z.string(),
  title: z.string(),
  description: z.string().optional(),
  content: z.string(),
  href: z.string(),
  _snippetResult: z
    .object({
      title: z
        .object({
          value: z.string(),
        })
        .optional(),
      content: z
        .object({
          value: z.string(),
        })
        .optional(),
    })
    .optional(),
});

const hitsSchema = hitSchema.array();

const search = async (term: string) => {
  const res = await fetch(ALGOLIA_URL, {
    method: "POST",
    headers: ALGOLIA_HEADERS,
    body: JSON.stringify({
      params: `query=${term}&hitsPerPage=5&attributesToSnippet=content,title:20&highlightPreTag=<b>&highlightPostTag=</b>`,
    }),
  });
  const json = await res.json();
  const hits = hitsSchema.safeParse(json.hits);
  if (!hits.success) {
    console.error(hits.error.issues);
    return [];
  }

  return hits.data.map((hit) => ({
    key: hit.objectID,
    title: hit._snippetResult?.title?.value ?? hit.title,
    description: hit.description,
    content: hit._snippetResult?.content?.value ?? hit.content,
    href: hit.href,
  })) as SearchResult[];
};

const DEFAULT_ITEMS: SearchResult[] = [
  {
    key: "default-get-started",
    title: "Get started",
    description: "Get started with Synnax",
    content: "Install Synnax and learn how it organizes your data",
    href: "/reference",
  },
  {
    key: "default-channels",
    title: "Channels",
    description: "Store and stream sensor data",
    content: "Learn how channels hold every value that Synnax records",
    href: "/reference/channels",
  },
  {
    key: "default-hardware",
    title: "Hardware",
    description: "Acquire data from devices",
    content: "Connect NI, LabJack, OPC UA, Modbus, EtherCAT, and HTTP devices",
    href: "/reference/hardware",
  },
  {
    key: "default-visualize",
    title: "Visualize",
    description: "Plot and monitor data",
    content: "Build line plots, schematics, tables, and logs",
    href: "/reference/visualize/line-plots",
  },
  {
    key: "default-enterprise",
    title: "Set up Enterprise",
    description: "Deploy a Synnax Core",
    content: "Deploy and configure a Core for your team",
    href: "/reference/enterprise",
  },
];

const SearchDialogContent = () => {
  const { close, visible } = Dialog.useContext();
  const [value, setValue] = useState<string>("");
  const inputRef = useRef<HTMLInputElement>(null);
  const [data, setData] = useState<SearchResult[]>([]);
  const handleSearch = useCallback(
    (query: string) => {
      setValue(query);
      if (query.length === 0) return;
      void search(query)
        .then((data) => {
          setData(data);
        })
        .catch(console.error);
    },
    [visible],
  );
  const displayData = value.length === 0 ? DEFAULT_ITEMS : data;
  const { data: items, getItem } = List.useStaticData<string, SearchResult>({
    data: displayData,
  });
  return (
    <Select.Frame<string, SearchResult>
      data={items}
      getItem={getItem}
      value=""
      onChange={(k: string | null) => {
        if (k == null) return;
        document.getElementById(k)?.click();
        close();
      }}
    >
      <Input.Text
        className="search-results__input"
        ref={inputRef}
        placeholder={
          <>
            <Icon.Search />
            Search
          </>
        }
        borderColor={6}
        autoFocus
        value={value}
        onChange={handleSearch}
        size="huge"
        full="x"
      />
      <List.Scroll className="styled-scrollbar" background={0} bordered borderColor={6}>
        <List.Items<string, SearchResult>
          emptyContent={
            <Text.Text center status="disabled">
              No results
            </Text.Text>
          }
        >
          {searchListItem}
        </List.Items>
      </List.Scroll>
    </Select.Frame>
  );
};
