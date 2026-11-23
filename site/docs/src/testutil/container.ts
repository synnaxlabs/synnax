// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import reactRenderer from "@astrojs/react/server.js";
import { experimental_AstroContainer } from "astro/container";

/**
 * Creates a container that renders React components and the `client:interaction`
 * islands of the docs.
 */
export const createContainer = async (
  site?: string,
): Promise<experimental_AstroContainer> => {
  // The container takes directives only from a manifest, whose type has every field,
  // and it fills the fields that a partial manifest leaves out. The server never runs
  // the directive's script, so a stub stands in for it.
  const options = {
    astroConfig: { site },
    manifest: { clientDirectives: new Map([["interaction", "/* interaction */"]]) },
  };
  const container: unknown = await Reflect.apply(
    experimental_AstroContainer.create.bind(experimental_AstroContainer),
    undefined,
    [options],
  );
  if (!(container instanceof experimental_AstroContainer))
    throw new Error("the container failed to start");
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  return container;
};
