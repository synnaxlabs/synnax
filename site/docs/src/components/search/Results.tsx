// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement, useCallback, useMemo, useState } from "react";

import { Nav } from "@/components/nav";
import {
  type Entry,
  fromHit,
  group,
  hitSchema,
  type Result,
} from "@/components/search/result";
import { Text as DocsText } from "@/components/text";

const ALGOLIA_APP_ID = "YWD9T0JXCS";
const ALGOLIA_SEARCH_ONLY_API_KEY = "1f8b0497301392c94adedf89a98afb6f";
const ALGOLIA_URL = `https://${ALGOLIA_APP_ID}-dsn.algolia.net/1/indexes/docs_site/query`;
const ALGOLIA_HEADERS = {
  "X-Algolia-API-Key": ALGOLIA_SEARCH_ONLY_API_KEY,
  "X-Algolia-Application-Id": ALGOLIA_APP_ID,
};

// A result shows the icon of the deepest nav group that has one, such as a vendor under
// Hardware.
const ICONS: Record<string, Icon.ReactElement> = {
  ...Nav.SECTION_ICONS,
  blog: <Icon.Guide />,
  releases: <Icon.Release />,
  ni: <Icon.Logo.NI />,
  labjack: <Icon.Logo.LabJack />,
  modbus: <Icon.Logo.Modbus />,
  "opc-ua": <Icon.Logo.OPCUA />,
  ethercat: <Icon.Logo.EtherCAT />,
  http: <Icon.Logo.HTTP />,
  python: <Icon.Python />,
  arc: <Icon.Arc />,
};

const Item = (props: List.ItemRenderProps<string>) => {
  const { itemKey } = props;
  const item = List.useItem<string, Entry>(itemKey);
  if (item == null) return null;
  const { href, title, excerpt, section, path, ancestors, first } = item;
  const icon = ancestors.findLast((key) => key in ICONS);
  return (
    <>
      {first && (
        <Text.Text className="search-results__section" level="small">
          {section}
        </Text.Text>
      )}
      <Select.Item<string, "a">
        className="search-result"
        id={itemKey}
        el="a"
        href={href}
        propagateClick
        {...props}
      >
        <span className="search-result__icon">
          {icon == null ? <Icon.Reference /> : ICONS[icon]}
        </span>
        <span
          className="search-result__title"
          dangerouslySetInnerHTML={{ __html: DocsText.codifyHTML(title) }}
        />
        {path.length > 0 && (
          <span className="search-result__path">
            {path.map((name) => (
              <span key={name}>
                <DocsText.InlineCode text={name} />
              </span>
            ))}
          </span>
        )}
        <span
          className="search-result__excerpt"
          dangerouslySetInnerHTML={{ __html: DocsText.codifyHTML(excerpt) }}
        />
        <Triggers.Text
          className="search-result__enter"
          trigger={["Enter"]}
          level="small"
        />
      </Select.Item>
    </>
  );
};

const item = Component.renderProp(Item);

const hitsSchema = hitSchema.array();

const PARAMS = new URLSearchParams({
  hitsPerPage: "8",
  attributesToRetrieve: "title,href,description",
  attributesToSnippet: "content:14,title:20",
  highlightPreTag: "<b>",
  highlightPostTag: "</b>",
  snippetEllipsisText: "…",
});

const search = async (term: string): Promise<Result[]> => {
  const params = new URLSearchParams(PARAMS);
  params.set("query", term);
  const res = await fetch(ALGOLIA_URL, {
    method: "POST",
    headers: ALGOLIA_HEADERS,
    body: JSON.stringify({ params: params.toString() }),
  });
  const json = await res.json();
  const hits = hitsSchema.safeParse(json.hits);
  if (!hits.success) {
    console.error(hits.error.issues);
    return [];
  }
  return hits.data.map(fromHit);
};

const SUGGESTED: Result[] = [
  {
    key: "suggested-get-started",
    title: "Get started",
    excerpt: "Install Synnax and learn how it organizes your data",
    href: "/reference",
    ancestors: ["get-started"],
  },
  {
    key: "suggested-channels",
    title: "Channels",
    excerpt: "Learn how channels hold every value that Synnax records",
    href: "/reference/channels",
    ancestors: ["channels"],
  },
  {
    key: "suggested-hardware",
    title: "Hardware",
    excerpt: "Connect NI, LabJack, OPC UA, Modbus, EtherCAT, and HTTP devices",
    href: "/reference/hardware",
    ancestors: ["hardware"],
  },
  {
    key: "suggested-visualize",
    title: "Visualize",
    excerpt: "Build line plots, schematics, tables, and logs",
    href: "/reference/visualize",
    ancestors: ["visualize"],
  },
  {
    key: "suggested-enterprise",
    title: "Set up Enterprise",
    excerpt: "Deploy and configure a Core for your team",
    href: "/reference/enterprise",
    ancestors: ["enterprise"],
  },
].map((result) => ({ ...result, section: "Suggested", path: [] }));

/** Searches the docs as the reader types and opens the chosen page. */
export const Results = (): ReactElement => {
  const { close, visible } = Dialog.useContext();
  const [value, setValue] = useState<string>("");
  const [data, setData] = useState<Result[]>([]);
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
  const empty = value.length === 0;
  const entries = useMemo(() => group(empty ? SUGGESTED : data), [empty, data]);
  const { data: items, getItem } = List.useStaticData<string, Entry>({
    data: entries,
  });
  return (
    <Select.Frame<string, Entry>
      data={items}
      getItem={getItem}
      value=""
      initialHover={0}
      onChange={(k: string | null) => {
        if (k == null) return;
        document.getElementById(k)?.click();
        close();
      }}
    >
      <Input.Text
        className="search-results__input"
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
      <List.Scroll
        className="styled-scrollbar search-results__list"
        background={0}
        bordered
        borderColor={6}
      >
        <List.Items<string, Entry>
          emptyContent={
            <Text.Text center status="disabled">
              No results
            </Text.Text>
          }
        >
          {item}
        </List.Items>
      </List.Scroll>
      <Flex.Box
        className="search-results__footer"
        x
        align="center"
        gap="large"
        bordered
        borderColor={6}
      >
        <Triggers.Text trigger={["ArrowUp", "ArrowDown"]} level="small">
          Move
        </Triggers.Text>
        <Triggers.Text trigger={["Enter"]} level="small">
          Open
        </Triggers.Text>
        <Triggers.Text trigger={["Escape"]} level="small">
          Close
        </Triggers.Text>
      </Flex.Box>
    </Select.Frame>
  );
};
