// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, List, Select, Text } from "@synnaxlabs/pluto";
import { type MarkdownHeading } from "astro";
import { unescape } from "html-escaper";
import {
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

const ON_THIS_PAGE_ID = "on-this-page-heading";

export interface ReleaseListProps {
  headings?: MarkdownHeading[];
}

interface KeyedHeading extends Omit<MarkdownHeading, "slug"> {
  key: string;
}

export const ReleaseList = ({
  headings = [],
}: ReleaseListProps): ReactElement | null => {
  const data: KeyedHeading[] = useMemo(
    () =>
      headings.map(({ slug, ...h }) => ({
        ...h,
        key: slug,
      })),
    [headings],
  );
  const toc = useRef<HTMLDivElement>(null);
  const [currentID, setCurrentID] = useState("");

  useEffect(() => {
    if (toc.current == null) return;

    const setCurrent: IntersectionObserverCallback = (entries) => {
      for (const entry of entries)
        if (entry.isIntersecting) {
          const { id } = entry.target;
          if (id === ON_THIS_PAGE_ID) continue;
          setCurrentID(entry.target.id);
          break;
        }
    };

    const observerOptions: IntersectionObserverInit = {
      // Negative top margin accounts for `scroll-margin`. Negative bottom margin means
      // heading needs to be towards top of viewport to trigger intersection.
      rootMargin: "-50px 0% -66%",
      threshold: 0.15,
    };

    const headingsObserver = new IntersectionObserver(setCurrent, observerOptions);

    // Observe all the headings in the main page content.
    document
      .querySelectorAll("article :is(h6)")
      .forEach((h) => headingsObserver.observe(h));

    // Stop observing when the component is unmounted.
    return () => headingsObserver.disconnect();
  }, [toc.current]);

  // If there are no headings, return an empty div.
  if (headings.length === 0) return null;

  const getItem = useCallback(
    (keys: string | string[]) => {
      if (typeof keys === "string") return data.find((h) => h.key === keys);
      return data.filter((h) => keys.includes(h.key));
    },
    [data],
  ) as List.GetItem<string, KeyedHeading>;

  return (
    <Flex.Box el="nav" className="release-list" gap={2}>
      <Text.Text level="h5">History</Text.Text>
      <div ref={toc}>
        <Select.Frame<string, KeyedHeading>
          data={data.map((h) => h.key)}
          onChange={(keys: string | string[]) => {
            if (typeof keys === "string") setCurrentID(keys);
            else setCurrentID(keys[0]);
          }}
          getItem={getItem}
        >
          <List.Items<string, KeyedHeading>>
            {(p) => {
              const { getItem } = List.useUtilContext<string, KeyedHeading>();
              const heading = getItem?.(p.key);
              if (heading == null) return null;
              return (
                <Select.ListItem
                  href={`#${heading.key}`}
                  level="small"
                  index={p.index}
                  selected={currentID === heading.key}
                  key={heading.key}
                  itemKey={heading.key}
                  id={heading.key}
                  onClick={() => setCurrentID(heading.key)}
                  className={`header-link ${heading.key} depth-${heading.depth}`.trim()}
                >
                  {unescape(heading.text)}
                </Select.ListItem>
              );
            }}
          </List.Items>
        </Select.Frame>
      </div>
    </Flex.Box>
  );
};
