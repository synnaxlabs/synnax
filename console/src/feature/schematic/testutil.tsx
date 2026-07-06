// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, project, type schematic } from "@synnaxlabs/client";
import { Haul, Schematic as PSchematic, Triggers } from "@synnaxlabs/pluto";
import { type aether } from "@synnaxlabs/pluto/ether";
import { id } from "@synnaxlabs/x";
import { act, render, screen, within } from "@testing-library/react";
import {
  type ComponentType,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  Suspense,
} from "react";

import { Schematic } from "@/feature/schematic";
import { Modals } from "@/platform/modals";
import { Ontology } from "@/platform/ontology";
import { createServices } from "@/platform/ontology/testutil";
import { Session } from "@/session";
import {
  type ConsolePreloadedState,
  createConsoleWrapper,
  uniqueName,
} from "@/testutil";

export const client = createTestClient();

let projectKey: string | undefined;

/** Returns the key of the shared test project every created schematic lives under. */
export const testProjectKey = async (): Promise<string> =>
  (projectKey ??= (await client.projects.create({ name: id.create(), layout: {} }))
    .key);

/** Creates a schematic on the test cluster under a shared test project. */
export const createSchematic = async (
  overrides: Partial<schematic.New> = {},
): Promise<schematic.Schematic> =>
  await client.schematics.create(await testProjectKey(), {
    name: uniqueName("schematic"),
    ...overrides,
  });

/**
 * Builds preloaded Session state that registers per-schematic UI state under key,
 * merged over the schematic zero state.
 */
export const createPreloadedState = (
  key: string,
  overrides: Partial<Session.Schematic.State> = {},
): ConsolePreloadedState => ({
  [Session.Schematic.SLICE_NAME]: {
    ...Session.Schematic.ZERO_SLICE_STATE,
    schematics: { [key]: { ...Session.Schematic.ZERO_STATE, ...overrides } },
  },
});

// loadSchematic primes key's flux cache through the production retrieve path. The
// single-hook bootstrap keeps the suspending useEnsureRetrieved from being followed by
// other hooks, a shape that trips a React 19 concurrent-replay error.
const loadSchematic = async (
  Wrapper: FC<PropsWithChildren>,
  key: string,
): Promise<void> => {
  const Bootstrap = (): ReactElement => {
    PSchematic.useEnsureRetrieved({ key });
    return <div data-testid="loaded" />;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Suspense fallback={null}>
        <Bootstrap />
      </Suspense>,
      { wrapper: Wrapper },
    );
  });
  await within(utils.container).findByTestId("loaded");
};

export interface RenderSchematicOptions {
  schematic?: Partial<schematic.New>;
  sessionState?: Partial<Session.Schematic.State>;
  /** Extra aether components merged over the default console test registry. */
  additionalRegistry?: aether.ComponentRegistry;
}

/**
 * renderSchematic creates a schematic on the server, mounts Component with the
 * schematic loaded into the flux cache and its Session state preloaded, and returns
 * the render result plus the Redux store and schematic.
 */
export const renderSchematic = async (
  Component: ComponentType<{ layoutKey: string }>,
  {
    schematic: overrides,
    sessionState,
    additionalRegistry,
  }: RenderSchematicOptions = {},
) => {
  const created = await createSchematic(overrides);
  const { wrapper: Wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: createPreloadedState(created.key, sessionState),
    additionalRegistry,
  });
  await loadSchematic(Wrapper, created.key);
  const result = render(
    <PSchematic.Scope.Provider value={created.key}>
      <Component layoutKey={created.key} />
      <Modals.Stack />
    </PSchematic.Scope.Provider>,
    { wrapper: Wrapper },
  );
  return { key: created.key, schematic: created, result, store };
};

/**
 * renderSchematicTree creates a schematic on the server under a fresh project and
 * mounts the real ontology tree rooted at that project with the schematic ontology
 * service registered, plus a live modal stack. Returns the created schematic, its
 * tree root, and the backing Redux store.
 */
export const renderSchematicTree = async (overrides: Partial<schematic.New> = {}) => {
  const proj = await client.projects.create({ name: id.create(), layout: {} });
  const created = await client.schematics.create(proj.key, {
    name: uniqueName("schematic"),
    ...overrides,
  });
  const rootID = project.ontologyID(proj.key);
  const { wrapper: Console, store } = await createConsoleWrapper({ client });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Triggers.Provider>
        <Haul.Provider>{children}</Haul.Provider>
      </Triggers.Provider>
    </Console>
  );
  const services = createServices({ schematic: Schematic.ONTOLOGY_SERVICE });
  const result = render(
    <Ontology.ServicesProvider services={services}>
      <Ontology.Tree root={rootID} />
      <Modals.Stack />
    </Ontology.ServicesProvider>,
    { wrapper: Wrapper },
  );
  return { schematic: created, rootID, result, store };
};

/**
 * Finds the action button inside the symbol-edit sidebar section header with the
 * given title (e.g. the add button of the "Colors" or "Handles" section).
 */
export const getEditSectionHeaderButton = (title: string): HTMLButtonElement => {
  const header = screen.getByText(title).closest("header");
  const button = header?.querySelector("button");
  if (button == null) throw new Error(`no button in section header ${title}`);
  return button;
};

/** Returns the symbol-edit preview overlay handles with the given orientation. */
export const getOverlayHandles = (
  container: ParentNode,
  orientation: string,
): HTMLElement[] =>
  Array.from(
    container.querySelectorAll<HTMLElement>(`[data-orientation='${orientation}']`),
  );

/**
 * Returns the symbol-edit preview wrapper element carrying the inline pan/zoom
 * transform style.
 */
export const getPreviewTransformWrapper = (container: ParentNode): HTMLElement => {
  const el = container.querySelector<HTMLElement>("[style*='transform']");
  if (el == null) throw new Error("preview transform wrapper not found");
  return el;
};

/** The prompt text of the symbol-edit modal's initial SVG file-drop stage. */
export const SYMBOL_FILE_DROP_PROMPT =
  "Click to select an SVG file or drag and drop it here";

/** A minimal SVG whose single rect can act as a region selector target. */
export const SYMBOL_SVG =
  '<svg viewBox="0 0 10 10"><rect id="body" width="10" height="10" fill="#ffffff"/></svg>';

/** Builds a valid symbol wire payload for import fixtures. */
export const createSymbolPayload = (name: string) => ({
  name,
  data: {
    svg: SYMBOL_SVG,
    variant: "static",
    states: [{ key: "base", name: "Base", regions: [] }],
    handles: [],
    previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
  },
});
