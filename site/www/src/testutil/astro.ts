// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import reactRenderer from "@astrojs/react/server.js";
import {
  type ContainerRenderOptions,
  experimental_AstroContainer as AstroContainer,
} from "astro/container";
import { JSDOM } from "jsdom";

import config from "../../astro.config";

type Component = Parameters<AstroContainer["renderToString"]>[0];

/** Renders an Astro component on the server, as the build does, and parses it. */
export const renderAstro = async (
  component: Component,
  options?: ContainerRenderOptions,
): Promise<Document> => {
  const container = await AstroContainer.create({ astroConfig: { site: config.site } });
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  container.addClientRenderer({
    name: "@astrojs/react",
    entrypoint: "@astrojs/react/client.js",
  });
  const html = await container.renderToString(component, options);
  return new JSDOM(html).window.document;
};

// Astro tags each serialized prop as [type, value]: 0 for a plain value or object, 1
// for an array.
const decode = (value: unknown): unknown => {
  if (!Array.isArray(value))
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>).map(([k, v]) => [k, decode(v)]),
    );
  const [type, inner] = value as [number, unknown];
  if (type === 1) return (inner as unknown[]).map(decode);
  if (type !== 0) throw new Error(`unsupported island prop type ${type}`);
  return typeof inner === "object" && inner != null ? decode(inner) : inner;
};

/** Returns the props that the server passed to the island of the given component. */
export const islandProps = <P>(doc: Document, component: string): P => {
  const island = doc.querySelector(`astro-island[component-url="${component}"]`);
  if (island == null) throw new Error(`no island for ${component}`);
  return decode(JSON.parse(island.getAttribute("props")!)) as P;
};
