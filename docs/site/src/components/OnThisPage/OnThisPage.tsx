// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/pluto/menu";
import { type MarkdownHeading } from "astro";
import { unescape } from "html-escaper";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { Client } from "@/components/client";
import { Platform } from "@/components/platform";

const ON_THIS_PAGE_ID = "on-this-page-heading";

export interface OnThisPageProps {
  headings?: MarkdownHeading[];
  clients?: Client.Client[];
  platforms?: Platform.Platform[];
  url: string;
}

export const OnThisPage = ({
  headings = [],
  clients = [],
  platforms = [],
}: OnThisPageProps): ReactElement | null => {
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
      // Negative top margin accounts for `scroll-margin`.
      // Negative bottom margin means heading needs to be towards top of viewport to trigger intersection.
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

  // If there are no headings,
  // return an empty div.
  if (headings.length === 0) return null;

  return (
    <>
      {platforms.length > 0 && <Platform.SelectButton platforms={platforms} />}
      {clients.length > 0 && <Client.SelectButton clients={clients} />}
      <div ref={toc} style={{ flexGrow: 1 }}>
        <Menu.Menu value={currentID}>
          {headings
            .filter(({ depth }) => depth > 1 && depth <= 3)
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
                overflow="wrap"
                className={`header-link ${heading.slug} depth-${heading.depth} ${
                  currentID === heading.slug ? "current-header-link" : ""
                }`.trim()}
              >
                {unescape(heading.text)}
              </Menu.Item>
            ))}
        </Menu.Menu>
      </div>
    </>
  );
};
