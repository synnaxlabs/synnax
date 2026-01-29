// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MarkdownHeading } from "astro";
import { unescape } from "html-escaper";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { Client } from "@/components/client";
import { Platform } from "@/components/platform";

const ON_THIS_PAGE_ID = "on-this-page-heading";

interface IndicatorPosition {
  top: number;
  height: number;
}

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
  const menuRef = useRef<HTMLDivElement>(null);
  const [currentID, setCurrentID] = useState("");
  const [indicator, setIndicator] = useState<IndicatorPosition>({ top: 0, height: 0 });
  const [initialized, setInitialized] = useState(false);
  const [visibleHeadings, setVisibleHeadings] = useState<Set<string>>(
    () => new Set(headings.map(({ slug }) => slug)),
  );

  // Purge headings that don't exist in the DOM (hidden by tabs, etc.)
  useEffect(() => {
    const purge = () => {
      const titles = document.querySelectorAll("article :is(h1, h2, h3)");
      const visibleIds = new Set(Array.from(titles).map((t) => t.id));
      setVisibleHeadings(visibleIds);
    };
    purge();
    window.addEventListener("urlchange", purge);
    return () => window.removeEventListener("urlchange", purge);
  }, []);

  // Update indicator position when currentID changes
  useEffect(() => {
    if (!menuRef.current || !currentID) return;
    const activeItem = menuRef.current.querySelector(
      `[data-item-key="${currentID}"]`,
    ) as HTMLElement | null;
    if (activeItem) {
      setIndicator({
        top: activeItem.offsetTop,
        height: activeItem.offsetHeight,
      });
      setInitialized(true);
    }
  }, [currentID]);

  useEffect(() => {
    if (menuRef.current == null) return;

    const getTopHeading = (): string | null => {
      const headings = document.querySelectorAll("article :is(h1,h2,h3)");
      let topHeading: Element | null = null;
      let topDistance = Infinity;

      for (const heading of headings) {
        const rect = heading.getBoundingClientRect();
        const distance = Math.abs(rect.top - 120);
        if (rect.top <= 150 && rect.top > -rect.height && distance < topDistance) {
          topDistance = distance;
          topHeading = heading;
        }
      }

      if (!topHeading) {
        for (const heading of headings) {
          const rect = heading.getBoundingClientRect();
          if (rect.top < 150) {
            topHeading = heading;
          }
        }
      }

      return topHeading?.id || null;
    };

    const handleScroll = () => {
      const id = getTopHeading();
      if (id && id !== ON_THIS_PAGE_ID) {
        setCurrentID(id);
      }
    };

    handleScroll();
    window.addEventListener("scroll", handleScroll, { passive: true });

    return () => window.removeEventListener("scroll", handleScroll);
  }, [menuRef.current]);

  if (headings.length === 0) return null;

  const filteredHeadings = headings.filter(
    ({ depth, slug }) => depth > 1 && depth <= 3 && visibleHeadings.has(slug),
  );

  return (
    <>
      {platforms.length > 0 && <Platform.SelectButton platforms={platforms} />}
      {clients.length > 0 && <Client.SelectButton clients={clients} />}
      <div ref={menuRef} className="on-this-page-menu" style={{ flexGrow: 1 }}>
        <div
          className="on-this-page-indicator"
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
            className={`on-this-page-item depth-${heading.depth} ${currentID === heading.slug ? "active" : ""}`}
          >
            {unescape(heading.text)}
          </a>
        ))}
      </div>
    </>
  );
};
