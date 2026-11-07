// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/tree/search.css";

import { group, type ontology, resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { type Tree as Base } from "@synnaxlabs/lyra/tree";
import { type Flux, Group } from "@synnaxlabs/pluto";
import { type state } from "@synnaxlabs/x";
import {
  createContext,
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  type RefObject,
  use,
} from "react";

import { CSS } from "@/platform/css";
import { Toolbar } from "@/platform/toolbar";
import { toResource } from "@/platform/tree/children";

/** How a search shows its matches: in one list, or inside their groups. */
export type SearchLayout = "flat" | "groups";

/** What a search looks for. */
export interface SearchQuery {
  term: string;
  /** The tree root. A match outside it does not show. */
  root: resource.ID;
  /** The most matches a source returns. */
  limit: number;
}

/** The matches one source found, in rank order. */
export type Matches =
  | { variant: "loading" }
  | { variant: "success"; resources: ontology.Resource[] }
  | { variant: "error"; description?: string };

/**
 * Returns the matches of each source. The results show the sources in the order
 * given. A null query asks for nothing.
 */
export interface UseMatches {
  (query: SearchQuery | null): Matches[];
}

/** Converts a Flux result into the matches of one source. */
export const toMatches = <T extends state.State>(
  result: Flux.Result<T[]>,
  convert: (item: T) => ontology.Resource,
): Matches => {
  if (result.variant === "success")
    return { variant: "success", resources: result.data.map(convert) };
  if (result.variant === "error")
    return { variant: "error", description: result.status.description };
  return { variant: "loading" };
};

/**
 * Matches groups by name. A root that names a type keeps the groups of that scope; the
 * search places the matches and drops the ones outside the root.
 */
export const useGroupMatches: UseMatches = (query) => {
  const result = Group.useResultList(
    query == null
      ? null
      : {
          searchTerm: query.term,
          limit: query.limit,
          ...(query.root.key === "" ? { scope: query.root.type } : {}),
        },
  );
  return [toMatches(result, toResource("group"))];
};

const groupOf = ({ id, data }: ontology.Resource): group.Key | undefined => {
  if (id.type !== "group") return group.keyZ.optional().parse(data?.group);
  const parent = resource.idZ.optional().parse(data?.parent);
  return parent?.type === "group" ? parent.key : undefined;
};

/** Names what a result row shows of a match, so a change to it rebuilds the results. */
export const matchKey = (r: ontology.Resource): string =>
  [r.key, r.name, groupOf(r) ?? ""].join("\t");

/** Returns the keys of the groups that hold the resources. */
export const groupsAbove = (resources: ontology.Resource[]): group.Key[] => [
  ...new Set(resources.map(groupOf).filter((key) => key != null)),
];

// Whether a resource with no group above it sits at the top of the tree under root.
const atTop = (r: ontology.Resource, root: resource.ID): boolean => {
  if (r.id.type !== "group") return root.key === "";
  const parent = resource.idZ.optional().parse(r.data?.parent);
  if (parent != null) return resource.idsEqual(parent, root);
  return root.key === "" && r.data?.scope === root.type;
};

/** Where the search places its matches. */
export interface Placement {
  /** The groups between the matches and the root. */
  resources: ontology.Resource[];
  /** The groups from the root down to the parent of each match, keyed by match key. */
  paths: Record<string, resource.ID[][]>;
}

/**
 * Places each resource under its groups, reading the group of a member and the parent
 * of a group. A resource outside the root gets no path.
 * @param groups holds every group above the resources.
 */
export const placeInGroups = (
  resources: ontology.Resource[],
  groups: group.Group[],
  root: resource.ID,
): Placement => {
  const byKey = new Map(groups.map((g) => [g.key, toResource("group")(g)]));
  const pathOf = (r: ontology.Resource): ontology.Resource[] | null => {
    const path: ontology.Resource[] = [];
    let top = r;
    for (let key = groupOf(r); key != null; key = groupOf(top)) {
      if (root.type === "group" && key === root.key) return path;
      const g = byKey.get(key);
      if (g == null) return null;
      path.unshift(g);
      top = g;
    }
    return atTop(top, root) ? path : null;
  };
  const paths: Record<string, resource.ID[][]> = {};
  const used = new Map<string, ontology.Resource>();
  resources.forEach((r) => {
    const path = pathOf(r);
    if (path == null) return;
    path.forEach((g) => used.set(g.key, g));
    paths[r.key] = [path.map(({ id }) => id)];
  });
  return { resources: [...used.values()], paths };
};

export interface CreateSearchParams {
  /** What the search finds, in the plural, such as "channels". */
  noun: string;
  /** Defaults to {@link useGroupMatches}. */
  useMatches?: UseMatches;
}

/** A fixed set of nodes that takes the place of the children of the root. */
export interface ResultNodes {
  nodes: Base.Node<string>[];
  /** Groups that hold only the matches below them, not every child. */
  fixed: Set<string>;
  /** Groups to open on arrival, so the matches inside them show. */
  expanded: string[];
  /** The groups above each node at the top of the results, nearest first. */
  paths: Map<string, resource.ID[]>;
  /** The resource of each node. */
  resources: Map<string, ontology.Resource>;
  /** The keys of every group above each match, over all of its paths. */
  ancestors: Map<string, Set<string>>;
}

/** What shows in place of the tree while a search has a term. */
export type SearchAnswer =
  | { variant: "error"; description?: string }
  | { variant: "success"; results: ResultNodes; match: Match; more: boolean };

/** The state of the search of one tree. */
export interface Search {
  /** What the search finds, in the plural, such as "channels". */
  noun: string;
  open: boolean;
  term: string;
  layout: SearchLayout;
  /** Whether matches are on their way. */
  loading: boolean;
  /** Null while the tree shows. */
  answer: SearchAnswer | null;
  start: () => void;
  close: () => void;
  setTerm: (term: string) => void;
  setLayout: (layout: SearchLayout) => void;
  /** Takes the focus back from the results on Escape. */
  fieldRef: RefObject<HTMLInputElement | null>;
  /** Takes the focus from the field on ArrowDown. */
  resultsRef: RefObject<HTMLDivElement | null>;
  /** Takes the focus when the search closes. */
  treeRef: RefObject<HTMLDivElement | null>;
}

/** Runs the search of the tree under root. A null root searches nothing. */
export interface UseSearch {
  (root: resource.ID | null): Search;
}

export interface SearchActionProps {
  search: Search;
}

/** The header action that opens the search. */
export const SearchAction = ({ search }: SearchActionProps): ReactElement => (
  <Toolbar.Action onClick={search.start} tooltip={`Search ${search.noun}`}>
    <Icon.Search />
  </Toolbar.Action>
);

export interface SearchFieldProps {
  search: Search;
}

/** Takes the place of the header title while the search is open. */
export const SearchField = ({ search }: SearchFieldProps): ReactElement => {
  const {
    term,
    setTerm,
    layout,
    setLayout,
    close,
    loading,
    noun,
    answer,
    fieldRef,
    resultsRef,
  } = search;
  const enterResults = (e: KeyboardEvent<HTMLInputElement>) => {
    // A key that is part of an IME word, such as the Enter that ends it, stays in the
    // field. Safari marks that Enter only with the keyCode.
    // eslint-disable-next-line @typescript-eslint/no-deprecated
    if (e.nativeEvent.isComposing || e.keyCode === 229) return;
    const results = resultsRef.current;
    if (results == null || answer?.variant !== "success") return;
    if (answer.results.nodes.length === 0) return;
    if (e.shiftKey || e.altKey || e.ctrlKey || e.metaKey) return;
    e.preventDefault();
    results.focus();
    // The focus comes back to a selected match. With none, the results get a Down key,
    // so they select the first match.
    if (results.querySelector('[aria-selected="true"]') != null) return;
    const down = {
      key: "ArrowDown",
      code: "ArrowDown",
      bubbles: true,
      cancelable: true,
    };
    results.dispatchEvent(new window.KeyboardEvent("keydown", down));
    // A real Down gets its own release. Enter does not, so the Down it sends gets one,
    // or the triggers keep Down held.
    if (e.key === "Enter")
      results.dispatchEvent(new window.KeyboardEvent("keyup", down));
  };
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "ArrowDown" || e.key === "Enter") return enterResults(e);
    if (e.key !== "Escape") return;
    e.stopPropagation();
    close();
  };
  return (
    <>
      <Input.Text
        ref={fieldRef}
        className={CSS.BE("tree-search", "field")}
        size="small"
        level="p"
        variant="text"
        full="x"
        grow
        autoFocus
        value={term}
        placeholder={`Search ${noun}`}
        onChange={setTerm}
        onKeyDown={handleKeyDown}
        startContent={loading ? <Icon.Loading color={9} /> : <Icon.Search color={9} />}
      />
      <Toolbar.Actions>
        <Toolbar.ToggleAction
          value={layout === "groups"}
          onChange={(groups) => setLayout(groups ? "groups" : "flat")}
          tooltip="Show in groups"
        >
          <Icon.TreeView />
        </Toolbar.ToggleAction>
        <Toolbar.Action onClick={close} tooltip="Close search">
          <Icon.Close />
        </Toolbar.Action>
      </Toolbar.Actions>
    </>
  );
};

/** What a result item shows beside its name. */
export interface Match {
  term: string;
  /** The name of the group to show beside each match, keyed by its ID. */
  groups: Map<string, string>;
}

const MatchContext = createContext<Match | null>(null);
MatchContext.displayName = "Tree.MatchContext";

export interface MatchProviderProps {
  value: Match;
  children: ReactNode;
}

export const MatchProvider = ({ value, children }: MatchProviderProps) => (
  <MatchContext value={value}>{children}</MatchContext>
);

/** Returns the term to tint in an item name, or undefined outside a search. */
export const useHighlight = (): string | undefined => use(MatchContext)?.term;

export interface MatchGroupProps {
  id: resource.ID;
}

/** Shows the group of a match in the flat layout. Renders nothing elsewhere. */
export const MatchGroup = ({ id }: MatchGroupProps): ReactElement | null => {
  const name = use(MatchContext)?.groups.get(resource.idToString(id));
  if (name == null) return null;
  return (
    <Text.Text
      level="small"
      color={8}
      overflow="fade"
      className={CSS.BE("tree-item", "group")}
    >
      {name}
    </Text.Text>
  );
};
