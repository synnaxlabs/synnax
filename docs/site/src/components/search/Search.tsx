// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  useState,
  useCallback,
  useEffect,
  useRef,
  type ReactElement,
  useLayoutEffect,
} from "react";

import { Icon } from "@synnaxlabs/media";
import { Align } from "@synnaxlabs/pluto/align";
import { Dropdown } from "@synnaxlabs/pluto/dropdown";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { Triggers } from "@synnaxlabs/pluto/triggers";

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

  useLayoutEffect(() => {
    if (!d.visible && value !== "") setValue("");
  }, [d.visible, value]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent): void => {
      if (e.key === "Escape") {
        d.close();
        inputRef.current?.blur();
      }
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  return (
    <Triggers.Provider>
      <Dropdown.Dialog {...d} className="search-box">
        <Input.Text
          ref={inputRef}
          placeholder={
            <Text.WithIcon level="small" startIcon={<Icon.Search />}>
              Search Synnax
            </Text.WithIcon>
          }
          value={value}
          onChange={(v: string) => {
            void handleSearch(v);
          }}
          onFocus={d.open}
          centerPlaceholder
        />
        <List.List
          data={results}
          emptyContent={
            <Align.Center style={{ height: 150 }}>
              <Text.Text level="small">
                {value.length === 0 ? "Type to search..." : "No Results"}
              </Text.Text>
            </Align.Center>
          }
        >
          <List.Hover />
          <List.Selector<string, SearchResult>
            value={[]}
            allowMultiple={false}
            onChange={(k: string) => document.getElementById(k)?.click()}
          />
          <List.Core<string, SearchResult>>
            {({ entry: { key, href, title, content }, hovered }) => (
              <Align.Space<"a">
                id={key.toString()}
                el="a"
                direction="y"
                size="small"
                className={`search-result ${hovered ? "hovered" : ""}`}
                aria-selected={true}
                onClick={d.close}
                href={href}
                key={key}
              >
                <Text.Text level="h5" dangerouslySetInnerHTML={{ __html: title }} />
                <Text.Text
                  level="small"
                  dangerouslySetInnerHTML={{ __html: content }}
                />
              </Align.Space>
            )}
          </List.Core>
        </List.List>
      </Dropdown.Dialog>
    </Triggers.Provider>
  );
};
