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
  const html = await container.renderToString(component, options);
  return new JSDOM(html).window.document;
};
