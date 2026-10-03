// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { type PageNavNode } from "@/components/nav/Page";
import { REFERENCE_PAGES } from "@/pages/_nav";
import { type Page, PAGES, POSTS } from "@/util/pages";
import { normalizeRoute } from "@/util/route";

export const prerender = true;

const INTRO = `# Synnax

> Synnax is an observability and control platform for hardware operations: one platform
> for real-time data acquisition, hardware control, visualization, and analysis.

Synnax has a Core (a time-series database and server), the Console (a desktop app for
live plots, schematics, and tables), the Driver (data acquisition and control for
National Instruments, LabJack, Modbus, OPC UA, EtherCAT, and HTTP devices), Arc (a
reactive language for control sequences, alarms, and calculated channels), and Python,
TypeScript, and C++ client libraries.

Every docs page is also served as Markdown at its URL plus \`.md\`.`;

const PAGES_BY_ROUTE = new Map(PAGES.map((page) => [page.route, page]));

interface Entry {
  page: Page;
  /** The nav groups above the page, e.g. "Arc > Concepts". */
  group?: string;
}

const entries = (nodes: PageNavNode[], group?: string): Entry[] =>
  nodes.flatMap((node) => {
    if (node.children != null)
      return entries(
        node.children,
        group == null ? node.name : `${group} > ${node.name}`,
      );
    if (node.href == null) return [];
    const page = PAGES_BY_ROUTE.get(normalizeRoute(node.href));
    if (page == null) throw new Error(`nav links to ${node.href}, not a docs page`);
    return [{ page, group }];
  });

/** Serves the docs index for LLMs, as specified at https://llmstxt.org. */
export const GET: APIRoute = ({ site }) => {
  const item = ({ page, group }: Entry): string => {
    const label = group == null ? page.title : `${group}: ${page.title}`;
    const link = `- [${label}](${new URL(`${page.route}.md`, site).href})`;
    return page.description == null ? link : `${link}: ${page.description}`;
  };
  const sections: Array<[string, Entry[]]> = [
    [
      "Get started",
      entries(REFERENCE_PAGES.filter(({ children }) => children == null)),
    ],
    ...REFERENCE_PAGES.filter(({ children }) => children != null).map(
      (node): [string, Entry[]] => [node.name, entries(node.children ?? [])],
    ),
    ["Blog", POSTS.map((page) => ({ page }))],
  ];
  const listed = new Set(
    sections.flatMap(([, list]) => list.map(({ page }) => page.route)),
  );
  const releases = PAGES_BY_ROUTE.get("/releases");
  const rest = PAGES.filter(({ route }) => !listed.has(route) && route !== "/releases");
  if (rest.length > 0) sections.push(["Other pages", rest.map((page) => ({ page }))]);
  const optional = [
    ...(releases == null ? [] : [item({ page: releases })]),
    "- [Website](https://www.synnaxlabs.com): Product overview and demo booking.",
  ];
  const body = [
    ...sections.map(([name, list]) => `## ${name}\n\n${list.map(item).join("\n")}`),
    `## Optional\n\n${optional.join("\n")}`,
  ];
  return new Response(`${INTRO}\n\n${body.join("\n\n")}\n`, {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
};
