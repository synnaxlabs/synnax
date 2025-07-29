// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb, Component, Dialog, Flux, Icon, Select } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { Triggers } from "@synnaxlabs/pluto/triggers";
import { caseconv, deep } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef, useState } from "react";
import z from "zod";

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

export const Search = (): ReactElement => {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent): void => {
      if (e.key === "Escape") close();
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        open();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);
  return (
    <Triggers.Provider>
      <Dialog.Frame variant="modal" className="search-box">
        <Dialog.Trigger
          startIcon={<Icon.Search />}
          variant="outlined"
          justify="center"
          size="large"
        >
          Search
        </Dialog.Trigger>
        <Dialog.Dialog>
          <SearchDialogContent />
        </Dialog.Dialog>
      </Dialog.Frame>
    </Triggers.Provider>
  );
};

const ICONS: Record<string, Icon.ReactElement> = {
  "python-client": <Icon.Python />,
  "typescript-client": <Icon.TypeScript />,
  cluster: <Icon.Cluster />,
  console: <Icon.Visualize />,
  concepts: <Icon.Concepts />,
  guides: <Icon.Guide />,
  "opc-ua": <Icon.Logo.OPC />,
  ni: <Icon.Logo.NI />,
  "device-drivers": <Icon.Device />,
  pluto: <Icon.Table />,
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
    <Select.ListItem<string, "a">
      id={itemKey}
      el="a"
      direction="y"
      style={{ padding: "2.5rem 3rem" }}
      gap="medium"
      aria-selected={true}
      href={href}
      {...props}
    >
      <Align.Space direction="y" empty>
        <Text.Text level="h4" dangerouslySetInnerHTML={{ __html: title }} />
        <Breadcrumb.Breadcrumb level="small" separator="/" icon={icon}>
          {path}
        </Breadcrumb.Breadcrumb>
      </Align.Space>
      <Text.Text level="small" dangerouslySetInnerHTML={{ __html: content }} />
    </Select.ListItem>
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

const SearchDialogContent = () => {
  const { close, visible } = Dialog.useContext();
  const [value, setValue] = useState<string>("");
  const inputRef = useRef<HTMLInputElement>(null);
  const [data, setData] = useState<SearchResult[]>([]);
  const handleSearch = useCallback(
    (query: string) => {
      setValue(query);
      void search(query)
        .then((data) => {
          setData(data);
        })
        .catch(console.error);
    },
    [visible],
  );
  useEffect(() => {
    if (visible) handleSearch("");
  }, [visible]);
  const { data: items, getItem } = List.useStaticData<string, SearchResult>({
    data: data ?? [],
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
      <Align.Pack className="search-results__content" direction="y">
        <Input.Text
          className="search-results__input"
          ref={inputRef}
          placeholder={
            <Text.WithIcon level="h2" startIcon={<Icon.Search />}>
              Search
            </Text.WithIcon>
          }
          autoFocus
          value={value}
          onChange={handleSearch}
          size="huge"
        />
        <List.Items<string, SearchResult>
          className="styled-scrollbar"
          background={0}
          bordered
          borderShade={6}
          emptyContent={
            <Align.Center style={{ height: "100%" }}>
              <Text.Text level="p" shade={11} weight={400}>
                {value.length === 0 ? "Type to search..." : "No Results"}
              </Text.Text>
            </Align.Center>
          }
        >
          {searchListItem}
        </List.Items>
      </Align.Pack>
    </Select.Frame>
  );
};
