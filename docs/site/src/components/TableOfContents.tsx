// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useRef, useState } from "react";

import { Space, Typography, Header, Menu } from "@synnaxlabs/pluto";
import { MarkdownHeading } from "astro";
import { unescape } from "html-escaper";

interface ItemOffset {
  id: string;
  topOffset: number;
}

export const TableOfContents = ({
  headings = [],
}: {
  headings: MarkdownHeading[];
}): JSX.Element => {
  const toc = useRef<HTMLDivElement | null>();
  const onThisPageID = "on-this-page-heading";
  const itemOffsets = useRef<ItemOffset[]>([]);
  const [currentID, setCurrentID] = useState("");
  useEffect(() => {
    const getItemOffsets = (): void => {
      const titles = document.querySelectorAll("article :is(h1, h2, h3, h4)");
      itemOffsets.current = Array.from(titles).map((title) => ({
        id: title.id,
        topOffset: title.getBoundingClientRect().top + window.scrollY,
      }));
    };

    getItemOffsets();
    window.addEventListener("resize", getItemOffsets);
    return () => {
      window.removeEventListener("resize", getItemOffsets);
    };
  }, []);

  useEffect(() => {
    if (toc.current == null) return;

    const setCurrent: IntersectionObserverCallback = (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          const { id } = entry.target;
          if (id === onThisPageID) continue;
          setCurrentID(entry.target.id);
          break;
        }
      }
    };

    const observerOptions: IntersectionObserverInit = {
      // Negative top margin accounts for `scroll-margin`.
      // Negative bottom margin means heading needs to be towards top of viewport to trigger intersection.
      rootMargin: "-100px 0% -66%",
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

  return (
    <Space
      style={{
        paddingLeft: "2rem",
        transition: "0.2s ease-in-out",
      }}
    >
      <Header id={onThisPageID} className="heading">
        <Header.Title level="h3">On this page</Header.Title>
      </Header>
      <div ref={toc}>
        <Menu selected={currentID}>
          {headings
            .filter(({ depth }) => depth > 1 && depth < 3)
            .map((heading) => (
              <Typography.Text.Link
                href={`#${heading.slug}`}
                level="p"
                key={heading.slug}
                className={`header-link depth-${heading.depth} ${
                  currentID === heading.slug ? "current-header-link" : ""
                }`.trim()}
              >
                {unescape(heading.text)}
              </Typography.Text.Link>
            ))}
        </Menu>
      </div>
    </Space>
  );
};
