// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Keeps the links that `selector` matches on the reader's place in the current page:
 * its path, its query (which holds the tab choices), and its anchor. Each link names
 * the origin of its version in `data-origin`.
 */
export const track = (selector: string): void => {
  const follow = (link: HTMLAnchorElement): void => {
    const { pathname, search, hash } = location;
    link.href = `${link.dataset.origin}${pathname}${search}${hash}`;
  };
  document.addEventListener("astro:page-load", () => {
    for (const link of document.querySelectorAll<HTMLAnchorElement>(selector))
      follow(link);
  });
  // The tab sync changes the query with no page load, so a link updates on use too.
  const use = ({ target }: Event): void => {
    const link =
      target instanceof Element ? target.closest<HTMLAnchorElement>(selector) : null;
    if (link != null) follow(link);
  };
  document.addEventListener("click", use, true);
  document.addEventListener("auxclick", use, true);
};
