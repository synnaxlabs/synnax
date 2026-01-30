// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Text } from "@synnaxlabs/pluto";
import { type MarkdownHeading } from "astro";
import { unescape } from "html-escaper";
import { type ReactElement, useEffect, useRef, useState } from "react";

const ON_THIS_PAGE_ID = "on-this-page-heading";

interface IndicatorPosition {
  top: number;
  height: number;
}

export const List = ({
  headings = [],
}: {
  headings?: MarkdownHeading[];
}): ReactElement | null => {
  const menuRef = useRef<HTMLDivElement>(null);
  const [currentID, setCurrentID] = useState("");
  const [indicator, setIndicator] = useState<IndicatorPosition>({ top: 0, height: 0 });
  const [initialized, setInitialized] = useState(false);

  // Update indicator position when currentID changes
  useEffect(() => {
    if (!menuRef.current || !currentID) return;
    const activeItem = menuRef.current.querySelector<HTMLElement>(
      `[data-item-key="${currentID}"]`,
    );
    if (activeItem != null) {
      setIndicator({
        top: activeItem.offsetTop,
        height: activeItem.offsetHeight,
      });
      setInitialized(true);
    }
  }, [currentID]);

  useEffect(() => {
    if (menuRef.current == null) return;

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
      // Negative top margin accounts for header.
      // Large negative bottom margin means the element needs to be near the top to trigger.
      rootMargin: "-100px 0% -85%",
      threshold: 0,
    };

    const headingsObserver = new IntersectionObserver(setCurrent, observerOptions);

    // Observe all the headings in the main page content.
    document
      .querySelectorAll("article :is(h6)")
      .forEach((h) => headingsObserver.observe(h));

    // Stop observing when the component is unmounted.
    return () => headingsObserver.disconnect();
  }, [menuRef.current]);

  if (headings.length === 0) return null;

  const filteredHeadings = headings.filter(({ depth }) => depth === 6);

  return (
    <Flex.Box el="nav" className="release-list" gap={2}>
      <Text.Text level="h5">History</Text.Text>
      <div ref={menuRef} className="release-menu">
        <div
          className="release-indicator"
          style={{
            transform: `translateY(${indicator.top}px)`,
            height: indicator.height,
            opacity: initialized ? 1 : 0,
          }}
        />
        {filteredHeadings.map((heading) => (
          <a
            href={`#${heading.slug}`}
            key={heading.slug}
            data-item-key={heading.slug}
            onClick={() => setCurrentID(heading.slug)}
            className={`release-item ${currentID === heading.slug ? "active" : ""}`}
          >
            {unescape(heading.text)}
          </a>
        ))}
      </div>
    </Flex.Box>
  );
};
