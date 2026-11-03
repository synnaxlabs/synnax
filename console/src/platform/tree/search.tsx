// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/tree/search.css";

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { type Tree as Base } from "@synnaxlabs/lyra/tree";
import { type Flux, Ontology } from "@synnaxlabs/pluto";
import { type state } from "@synnaxlabs/x";
import {
  createContext,
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  use,
} from "react";

import { CSS } from "@/platform/css";
import { Toolbar } from "@/platform/toolbar";

/** How a search shows its matches: in one list, or inside their groups. */
export type SearchLayout = "flat" | "groups";

/** What a search looks for. */
export interface SearchQuery {
  term: string;
  /** The tree root. A match outside it does not show. */
  root: ontology.ID;
  /** The folder types and the types the toolbar searches for. */
  types: ontology.ResourceType[];
  /** The most matches a source returns. */
  limit: number;
}

/** The matches one source found, in rank order. */
export type Matches =
  | { variant: "loading" }
  | { variant: "success"; ids: ontology.ID[] }
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
  toID: (item: T) => ontology.ID,
): Matches => {
  if (result.variant === "success")
    return { variant: "success", ids: result.data.map(toID) };
  if (result.variant === "error")
    return { variant: "error", description: result.status.description };
  return { variant: "loading" };
};

const resourceID = ({ id }: ontology.Resource): ontology.ID => id;

/** Matches the resources of the query types under the root by name. */
export const useOntologyMatches: UseMatches = (query) => {
  const result = Ontology.useResultResources(
    query == null
      ? null
      : {
          searchTerm: query.term,
          root: query.root,
          types: query.types,
          limit: query.limit,
        },
  );
  return [toMatches(result, resourceID)];
};

export interface CreateSearchParams {
  /** What the search finds, in the plural, such as "channels". */
  noun: string;
  /** The types the search finds along with the folders, such as "arc". */
  types?: ontology.ResourceType[];
  /** Defaults to {@link useOntologyMatches}. */
  useMatches?: UseMatches;
}

/** A fixed set of nodes that takes the place of the children of the root. */
export interface ResultNodes {
  nodes: Base.Node<string>[];
  /** Groups that hold only the matches below them, not every child. */
  fixed: Set<string>;
  /** Groups to open on arrival, so the matches inside them show. */
  expanded: string[];
  /** The parent of each node at the top of the results. */
  parents: Map<string, ontology.ID>;
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
}

/** Runs the search of the tree under root. A null root searches nothing. */
export interface UseSearch {
  (root: ontology.ID | null): Search;
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
  const { term, setTerm, layout, setLayout, close, loading, noun } = search;
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Escape") return;
    e.stopPropagation();
    close();
  };
  return (
    <>
      <Input.Text
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
  id: ontology.ID;
}

/** Shows the group of a match in the flat layout. Renders nothing elsewhere. */
export const MatchGroup = ({ id }: MatchGroupProps): ReactElement | null => {
  const name = use(MatchContext)?.groups.get(ontology.idToString(id));
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
