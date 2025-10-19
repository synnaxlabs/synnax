// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { List, Select } from "@synnaxlabs/pluto";
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

import { Platform } from "@/components/platform";

const ON_THIS_PAGE_ID = "on-this-page-heading";

export interface OnThisPageProps {
  headings?: MarkdownHeading[];
  platforms?: Platform.Platform[];
}

interface KeyedHeading extends Omit<MarkdownHeading, "slug"> {
  key: string;
}

export const OnThisPage = ({
  headings = [],
  platforms = [],
}: OnThisPageProps): ReactElement | null => {
  const headingData: KeyedHeading[] = useMemo(
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
    const purge = () => {
      const titles = document.querySelectorAll("article :is(h1, h2, h3)");
      const headerLinks = document.querySelectorAll(
        ".on-this-page .header-link",
      ) as unknown as HTMLElement[];
      headerLinks.forEach((link) => {
        // check if there's a matching title
        const title = Array.from(titles).find((title) => title.id === link.id);
        if (title == null)
          // set the link display to none
          link.style.display = "none";
        else link.style.display = "block";
      });
    };
    purge();
    const i = setInterval(purge, 200);
    return () => clearInterval(i);
  }, []);

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
      threshold: 1,
    };

    const headingsObserver = new IntersectionObserver(setCurrent, observerOptions);

    // Observe all the headings in the main page content.
    document
      .querySelectorAll("article :is(h1,h2,h3)")
      .forEach((h) => headingsObserver.observe(h));

    // Stop observing when the component is unmounted.
    return () => headingsObserver.disconnect();
  }, [toc.current]);

  // If there are no headings, return an empty div.
  if (headings.length === 0) return null;

  const getItem = useCallback(
    (keys: string | string[]) => {
      if (typeof keys === "string") return headingData.find((h) => h.key === keys);
      return headingData.filter((h) => keys.includes(h.key));
    },
    [headingData],
  ) as List.GetItem<string, KeyedHeading>;

  return (
    <>
      {platforms.length > 0 && <Platform.SelectButton platforms={platforms} />}
      <div ref={toc} style={{ flexGrow: 1 }}>
        <Select.Frame<string, KeyedHeading>
          data={headingData.map((h) => h.key)}
          getItem={getItem}
          onChange={(keys: string | string[]) => {
            if (typeof keys === "string") setCurrentID(keys);
            else setCurrentID(keys[0]);
          }}
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
    </>
  );
};
