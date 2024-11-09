// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align } from "@synnaxlabs/pluto/align";
import { Header } from "@synnaxlabs/pluto/header";
import { Menu } from "@synnaxlabs/pluto/menu";
import { type MarkdownHeading } from "astro";
import { unescape } from "html-escaper";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { OSSelectButton } from "@/components/platform/PlatformTabs";

const ON_THIS_PAGE_ID = "on-this-page-heading";

export const OnThisPage = ({
  headings = [],
}: {
  headings?: MarkdownHeading[];
  url: string;
}): ReactElement => {
  const toc = useRef<HTMLDivElement>(null);
  const [currentID, setCurrentID] = useState("");

  useEffect(() => {
    const i = setInterval(() => {
      const titles = document.querySelectorAll("article :is(h1, h2, h3)");
      const headerLinks = document.querySelectorAll(
        ".on-this-page .header-link",
      ) as unknown as HTMLElement[];
      headerLinks.forEach((link) => {
        // check if there's a matching title
        const title = Array.from(titles).find((title) => {
          return title.id === link.id;
        });
        if (title == null) {
          // set the link display to none
          link.style.display = "none";
        } else {
          link.style.display = "block";
        }
      });
    }, 200);
    return () => clearInterval(i);
  }, []);

  // useEffect(() => {
  //   const getItemOffsets = (): void => {
  //     const titles = document.querySelectorAll("article :is(h1, h2, h3)");
  //     const headerLinks = document.querySelectorAll(".on-this-page .header-link");

  //     itemOffsets.current = Array.from(titles).map((title) => ({
  //       id: title.id,
  //       topOffset: title.getBoundingClientRect().top + window.scrollY,
  //     }));
  //   };

  //   getItemOffsets();
  //   window.addEventListener("resize", getItemOffsets);
  //   return () => {
  //     window.removeEventListener("resize", getItemOffsets);
  //   };
  // }, []);

  useEffect(() => {
    if (toc.current == null) return;

    const setCurrent: IntersectionObserverCallback = (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          const { id } = entry.target;
          if (id === ON_THIS_PAGE_ID) continue;
          setCurrentID(entry.target.id);
          break;
        }
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
  if (headings.length === 0) return <></>;

  return (
    <Align.Space el="nav" className="on-this-page" size={2}>
      <Header.Header id={ON_THIS_PAGE_ID} className="heading" level="h4">
        <Header.Title>On this page</Header.Title>
      </Header.Header>
      <OSSelectButton />
      <div ref={toc}>
        <Menu.Menu value={currentID}>
          {headings
            .filter(({ depth }) => depth > 1 && depth <= 3)
            .map((heading) => {
              return (
                <Menu.Item.Link
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
                </Menu.Item.Link>
              );
            })}
        </Menu.Menu>
      </div>
    </Align.Space>
  );
};
