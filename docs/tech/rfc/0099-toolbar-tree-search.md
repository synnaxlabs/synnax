# 99 Toolbar tree search

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-03
- **Related**: [RFC 0005 - MVP](0005-ontology.md),
  [RFC 0047 - Client cache, unified reads, and Console session state](0047-client-cache-unified-reads-session-state.md),
  [RFC 0054 - Flux read unification](0054-flux-read-unification.md)

## 0 Summary

A search action in a resource toolbar header changes the title into a field. Matches
replace the tree. In the flat layout, each match shows the name of its group. In the
groups layout, the matches show inside their groups. Esc or close ends the search, and
the tree comes back as it was. The Channels toolbar turns it on first.

Search is a mode of the shared ontology tree (`console/src/platform/tree`). Each toolbar
gives its own match rule. The Core places each match in its groups: one ontology request
returns the ancestors of many resources, with the parent links between them.

## 1 Motivation

A prototype of the interaction was accepted. It has three defects that block a release:

- **It sends one request per group level per match.** The client cannot batch the
  requests. A parents answer for many IDs holds only resources
  (`core/pkg/api/ontology/ontology.go:89-91`), so it cannot tell which parent belongs to
  which match. 100 matches three levels deep cost about 300 requests.
- **It has no limit.** The term "t" returns every channel on each keystroke.
- **Results are weaker than the tree.** Rename fails, because the tinted name is not
  editable (`lyra/src/text/Editable.tsx:222-232`). Move and group actions use the tree
  root as the parent (`console/src/platform/tree/Tree.tsx:485-490` on `main`). A failed
  request leaves the spinner on, and the results do not update live.

## 2 Design

### 2.0 Ancestors

The ontology retrieve request gains `ancestors`. For each resource in the answer, the
Core walks the parent links up to the root. Each step is a lookup in memory
(`parentsByIndex`, `core/pkg/service/ontology/retrieve.go:222-241`). The response adds
every resource passed on the way and every parent link walked:

```go
RetrieveRequest struct {
    // ...existing fields
    Ancestors bool        `json:"ancestors" msgpack:"ancestors"`
    Root      ontology.ID `json:"root"      msgpack:"root"`
}
RetrieveResponse struct {
    Resources     []ontology.Resource     `json:"resources"     msgpack:"resources"`
    Ancestors     []ontology.Resource     `json:"ancestors"     msgpack:"ancestors"`
    Relationships []ontology.Relationship `json:"relationships" msgpack:"relationships"`
}
```

`Resources` keeps its meaning, so a search and its placement are one request. Links
carry every path of a resource with more than one parent (RFC 0005 §2.1).

The TS client adds `client.ontology.ancestors` with `retrieve`, `onChange`, and
`getCached`, on the same query space machinery as `children` and `parents`
(`dependentSurface`, `client/ts/src/ontology/client.ts:433-465`). It writes the links
into the relationship cache, and it watches that cache, so a move refetches the paths it
touches. Each answer holds the resources and the paths from the root to each ID.
`ontology.Cache.paths` and `ontology.Cache.ancestorKeys` walk the cached links, beside
`parentID` (`client/ts/src/ontology/store.ts:66-71`). The relationship set and delete
streams (`client.ts:208-211`) keep paths live. Group renames arrive through the resource
cache.

### 2.1 Scoped ontology search

With a search term set, the API ignores `types`, `limit`, and `offset`, and it returns
at most 100 matches (`core/pkg/api/ontology/ontology.go:99-114`,
`core/pkg/service/ontology/ontology.go:189-206`). The search now honors them:

- **`types`**: Only the services for those types search.
- **`root`**: Keeps the matches whose ancestors include `root`. Each search index drops
  the other matches before it picks the full matches and applies the limit, so a full
  match outside `root` never hides a partial match inside it.
- **`limit` and `offset`**: Page through the merged matches in rank order.

The command palette already sends `limit` and `offset`
(`console/src/platform/palette/List.tsx:65`), so its paging starts to work.

### 2.2 Channel search limit

When a range is active, the channel search puts every alias match ahead of the name
matches. The alias matches skip `limit`, `offset`, and the other filters
(`core/pkg/api/channel/channel.go:266-278, 316-324`). Alias and name matches become one
ranked set, and every filter, `offset`, and `limit` apply to that set.

### 2.3 Tree search mode

A toolbar declares its search at module scope, the same way `Tree.createItem` takes
`useOnSelect` and `Nav.Toolbar` takes `useVisible`. The default match rule is the scoped
ontology search of §2.1. Channels matches groups with it, and channels with the channel
search, so aliases under the active range match what the tree shows
(`console/src/feature/channel/tree.tsx:208-213` on `main`):

```tsx
const SEARCH = Tree.createSearch({ noun: "channels", useMatches: useChannelMatches });

const Content = () => {
  const search = Tree.useSearch(SEARCH);
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        {search.open ? <Tree.SearchField search={search} /> : <Title search={search} />}
      </Toolbar.Header>
      <Toolbar.Body>
        <Tree.Tree root={root} search={search} />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};
```

| Concern       | Behavior                                                                         |
| ------------- | -------------------------------------------------------------------------------- |
| Order         | Groups first, then the other matches, each in rank order                         |
| Limit         | Each source asks for 101. At most 100 show, with a closing line when more exist  |
| Flat layout   | Each match shows the name of its parent group, from its first path               |
| Groups layout | Path groups start open and hold only matches. A matched group shows all children |
| Many parents  | The groups layout shows a match under each parent. The flat layout shows it once |
| Actions       | Every tree action works. The parent is the last group of the match's path        |
| Live          | Matches refetch on change (RFC 0047 §5.1, rule 3). Paths follow §2.0             |
| Errors        | A failed search shows a status in place of the results                           |
| Close         | The browse tree stays mounted, hidden with a class, so it comes back as it was   |

The closing line reads "Showing the first 100 matches. Type more to narrow the search."
The error status has the message "Failed to search channels" and the error text as its
description. A failed live refetch keeps the current results
(`client/ts/src/query/query.ts:663-686`). Results keep the `.lyra-tree` container and
the item IDs, so the tree helpers in `integration/console/` work on them.

### 2.4 Tinted names and rename

Lyra `Text` takes a `highlight` term. It tints the substrings that match a term token,
with the tokens split the same way as the Core search (`x/go/search/token.go:23`). A
match within one edit shows no tint. `Text.Editable` shows the tint while you are not
editing, and plain text while you are, so rename works on a result.

## 3 Implementation phases

Phase 1 is the wire. It ships with two callers that need it today: `findProjectAncestor`
(`pluto/src/project/queries.ts:142-152`) becomes one ancestors request, not a recursive
walk. The `searchTerm` on `groups.fetchChildren` and `Group.useList` is deleted. The
server answered it with matches from every type, which the client then cached as
children (`client/ts/src/group/client.ts:184-199`). No Console code sends it.

Phase 2 is the whole interaction. It is large, near 900 lines with specs. A split has no
seam: the Lyra tint and the tree mode have no user without the Channels toolbar.

| Phase | Scope                                                | Touches                                                                                                                                                                              | Depends on |
| ----- | ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------- |
| 1     | Add ancestors and scoped search to ontology retrieve | `x/go/search`, `core/pkg/service/ontology`, `core/pkg/api/ontology`, `core/pkg/api/channel`, `client/ts/src/ontology`, `client/ts/src/group`, `pluto/src/group`, `pluto/src/project` | -          |
| 2     | Search the Channels toolbar tree                     | `lyra/src/text`, `pluto/src/ontology`, `console/src/platform/tree`, `console/src/feature/channel`, `docs/claude/ux.md`, `integration/console`                                        | 1          |

Every field is additive. A request that sent `types`, `limit`, or `offset` with a search
term now gets what it asked for. The palette is the only such caller.

## 4 Resolved decisions

1. **Search is a mode of the shared tree**: Channels, Devices, Projects, and Users
   render the same `Tree.Tree`, and placing a match in its groups is the same problem
   for each. A Channels-only build makes the next toolbar copy the path code.
2. **The Core returns ancestors as links**: The other option was a path option on each
   domain search, which every API and every client language would carry. The Console now
   makes one extra request for channel matches; the trade is real but small.
3. **Groups match by their own name**: Matching a channel by any group in its path needs
   a Core index of path text that changes on each move or group rename.
4. **The ontology search honors `types`, `limit`, `offset`, and `root`**: A
   parent-scoped search on the group API is smaller, but each later toolbar would need
   its own.
5. **Results support every tree action**: A weaker results mode is less code, but it
   makes a user close the search and find each match again in the tree.
6. **One page of 100, no infinite scroll**: People type more before they scroll that
   far. With infinite scroll, the groups layout jumps as pages arrive, and two sources
   need a merge. A search with more than 100 matches shows only part of them; the
   closing line says so.
7. **The browse tree stays mounted while a search shows**: A save and restore of the
   expanded groups, selection, and scroll position is more state for the same result.

## 5 Open questions

1. A shortcut to open the search. Nothing in a pane binds Ctrl+F or `/` today.
2. Whether the layout choice survives a reload, per toolbar.
