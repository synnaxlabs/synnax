// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, {
  useState,
  useCallback,
  useEffect,
  useRef,
  type ReactElement,
} from "react";

import { Icon } from "@synnaxlabs/media";
import { Align } from "@synnaxlabs/pluto/align";
import { Dropdown } from "@synnaxlabs/pluto/dropdown";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { Triggers } from "@synnaxlabs/pluto/triggers";
import { Button } from "@synnaxlabs/pluto/button";
import { Breadcrumb } from "@synnaxlabs/pluto";
import { caseconv, deep } from "@synnaxlabs/x";

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
  const d = Dropdown.use();
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent): void => {
      if (e.key === "Escape") {
        d.close();
      }
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        d.open();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);
  return (
    <Triggers.Provider>
      <Dropdown.Dialog variant="modal" zIndex={1000} {...d} className="search-box">
        <Button.Button
          startIcon={<Icon.Search />}
          onClick={d.open}
          variant="outlined"
          justify="center"
          size="small"
          endContent={<Triggers.Text level="small" trigger={["Control", "K"]} />}
        >
          Search
        </Button.Button>
        <SearchDialogContent d={d} />
      </Dropdown.Dialog>
    </Triggers.Provider>
  );
};

interface SearchDialogContentProps {
  d: Dropdown.DialogProps;
}

const ICONS: Record<string, ReactElement> = {
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
};

export const SearchListItem = (props: List.ItemProps<string, SearchResult>) => {
  const {
    entry: { key, href, title, content },
    hovered,
  } = props;
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
    <List.ItemFrame
      id={key.toString()}
      el="a"
      direction="y"
      style={{ padding: "2.5rem 3rem" }}
      size="medium"
      className={`search-result ${hovered ? "hovered" : ""}`}
      aria-selected={true}
      href={href}
      key={key}
      {...props}
    >
      <Align.Space direction="y" empty>
        <Text.Text level="h4" dangerouslySetInnerHTML={{ __html: title }} />
        <Breadcrumb.Breadcrumb level="small" separator="/" icon={icon}>
          {path}
        </Breadcrumb.Breadcrumb>
      </Align.Space>
      <Text.Text level="small" dangerouslySetInnerHTML={{ __html: content }} />
    </List.ItemFrame>
  );
};

const searchListItem = (props: List.ItemProps<string, SearchResult>) => (
  <SearchListItem {...props} />
);

const SearchDialogContent = ({ d }: SearchDialogContentProps) => {
  const [results, setResults] = useState<SearchResult[]>([]);
  const [value, setValue] = useState<string>("");
  const inputRef = useRef<HTMLInputElement>(null);
  const handleSearch = useCallback(async (query: string) => {
    setValue(query);
    const res = await fetch(ALGOLIA_URL, {
      method: "POST",
      headers: ALGOLIA_HEADERS,
      body: JSON.stringify({
        params: `query=${query}&hitsPerPage=5&attributesToSnippet=content,title:20&highlightPreTag=<b>&highlightPostTag=</b>`,
      }),
    });
    const json = await res.json();
    setResults(
      json.hits.map((hit: any) => ({
        key: hit.objectID,
        title: hit._snippetResult?.title?.value ?? hit.title,
        description: hit.description,
        content: hit._snippetResult.content.value,
        href: hit.href,
      })) as SearchResult[],
    );
  }, []);

  useEffect(() => {
    inputRef.current?.focus();
  }, [d.visible]);

  return (
    <List.List
      data={results}
      emptyContent={
        <Align.Center style={{ height: "100%" }}>
          <Text.Text level="p" shade={7} weight={400}>
            {value.length === 0 ? "Type to search..." : "No Results"}
          </Text.Text>
        </Align.Center>
      }
    >
      <List.Selector<string, SearchResult>
        value={[]}
        allowMultiple={false}
        onChange={(k: string) => {
          document.getElementById(k)?.click();
          d.close();
        }}
      >
        <List.Hover>
          <Align.Pack className="search-results__content" direction="y" borderShade={4}>
            <Input.Text
              ref={inputRef}
              placeholder={
                <Text.WithIcon level="small" startIcon={<Icon.Search />}>
                  Search
                </Text.WithIcon>
              }
              autoFocus
              value={value}
              onChange={(v: string) => {
                void handleSearch(v);
              }}
              size="large"
            />
            <List.Core<string, SearchResult> className="styled-scrollbar">
              {searchListItem}
            </List.Core>
          </Align.Pack>
        </List.Hover>
      </List.Selector>
    </List.List>
  );
};
