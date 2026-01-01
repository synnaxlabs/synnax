// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Menu, Text } from "@synnaxlabs/pluto";
import { type MarkdownHeading } from "astro";
import { unescape } from "html-escaper";
import { type ReactElement, useEffect, useRef, useState } from "react";

const ON_THIS_PAGE_ID = "on-this-page-heading";

export const ReleaseList = ({
  headings = [],
}: {
  headings?: MarkdownHeading[];
  url: string;
}): ReactElement | null => {
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
      // Negative top margin accounts for `scroll-margin`.
      // Negative bottom margin means heading needs to be towards top of viewport to trigger intersection.
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

  // If there are no headings,
  // return an empty div.
  if (headings.length === 0) return null;

  return (
    <Flex.Box el="nav" className="release-list" gap={2}>
      <Text.Text level="h5">History</Text.Text>
      <div ref={toc}>
        <Menu.Menu value={currentID}>
          {headings
            .filter(({ depth }) => depth === 6)
            .map((heading) => (
              <Menu.Item
                href={`#${heading.slug}`}
                level="small"
                key={heading.slug}
                itemKey={heading.slug}
                id={heading.slug}
                onClick={() => {
                  setCurrentID(heading.slug);
                }}
                className={`header-link ${heading.slug} depth-${heading.depth} ${
                  currentID === heading.slug ? "current-header-link" : ""
                }`.trim()}
              >
                {unescape(heading.text)}
              </Menu.Item>
            ))}
        </Menu.Menu>
      </div>
    </Flex.Box>
  );
};
