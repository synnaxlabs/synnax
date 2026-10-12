# React lenses

Run on every change that adds or modifies a component or hook. Run these in addition to
the nine base lenses. The repo does not use the React Compiler. Memoization is manual,
so every identity a hook or a prop creates is a decision.

1. **Hook discipline.** Is each new or changed component as lean in hooks as it can be?
   - Count the hooks before and after. Each one needs a job that nothing else can do.
   - Subscribe narrowly. Select the smallest slice of a store or context the component
     reads. Do not subscribe to a whole object to read one field.
   - `useMemo` and `useCallback` must protect a real consumer: a memoized child, a hook
     dependency, or an expensive computation. Otherwise they cost a hook and give
     nothing.
   - Lifecycle bookkeeping (a `useRef` flag, a `useEffect` cleanup, start and end
     branches) inside a consumer means the substrate is the wrong shape. The substrate
     should expose a hook that owns it.
2. **Prop discipline.** Do the props follow the canonical conventions?
   - Controlled values are `value` and `onChange`. Handlers are `on<Event>`. Booleans
     are adjectival predicates (`disabled`, not `disable` or `isDisabled`).
   - The namespace carries the context. Inside `Select`, the type is `Select.ItemProps`
     and its fields do not repeat "select".
   - A props type that copies fields from another component must extend it with `Pick`
     or `Omit` instead. `Omit<Flex.BoxProps, ...>` and `Pick<FrameProps, ...>` are the
     local patterns.
   - Do not add a prop for a variation that has only one consumer. Change the component.
   - Do not destructure a prop only to pass it through unchanged. Spread the rest
     object.
3. **Component architecture.** Does the API compose like blocks, in the style of Radix?
   - Prefer parts that nest (`Select.Frame`, `Select.Dialog`, `Select.Item`) over one
     component with a large config object or many slot props.
   - A render prop is acceptable only where it is the canonical choice, for example to
     render a data-driven item in a virtualized list. Otherwise it is a finding.
   - Shared state between parts goes through a context that the root part owns. It does
     not go through prop drilling or through cloning children.
   - Visual state that CSS can express (`:hover`, `:active`, `:focus-visible`, a
     modifier class) stays in CSS. It does not become React state.
4. **Leaky abstractions.** Does the new code depend on modules it should not know about?
   - Package direction: Lyra depends only on X. Pluto can use Lyra and the Client. The
     Console can use all of them. A primitive that imports upward is a defect.
   - Console layers: `session` < `platform` < `feature` < `app`. A feature never imports
     a sibling feature.
   - Worker safety: code on the Aether worker never imports a PascalCase (main-thread)
     namespace.
   - A generic component that knows about one domain type, the Redux store, or the
     Client is a leak. Pass the data in, or move the component up a layer.
   - Imports from another module go through its namespace, not its internal files.
5. **Render performance.** What does this change do to render cost and UI smoothness?
   - Find the frequent paths the change touches: each keystroke, each drag frame, each
     scroll, each streamed frame, each hover. For each one, say which components render
     again and how many.
   - Context values, inline objects, inline arrays, and inline `style={{...}}` make a
     new identity on each render. Each consumer and each memoized child then renders
     again.
   - In a list or a tree, a change to one item must not render all items again. Look for
     a render-count spec built on `createRenderCounter`
     (`lyra/src/testutil/renders.tsx`). If the change affects a large collection and has
     no such spec, that is a lens 8 finding.
   - Layout reads (`getBoundingClientRect`, `offsetWidth`) mixed with style writes in
     the same frame cause layout thrash. Animate with `transform` and `opacity`, not
     with layout properties.
6. **Side effects, data fetching, and queries.** Is each read, write, and effect on the
   right path, and is each effect necessary?
   - Core data goes through Flux. A read in render uses a query from the domain's
     `queries.ts` (`createRetrieve`, `createList`). A write uses `createUpdate`, and a
     form uses `createForm`. A `useEffect` that calls the Client and stores the answer
     in `useState` is a finding. If the domain has no query for the resource, add one to
     its `queries.ts`. Do not write an ad hoc fetch.
   - Use the correct read. `use` suspends and is the default. `createSelector` gives a
     narrow read that re-renders only on its slice, below a parent that called
     `useEnsure`. `useResult` is only for places where suspense is not permitted or
     where absence is a state to render. A read inside an event handler calls the Client
     directly. It is not a hook.
   - Count the requests. Look for fetches on each mount, a fetch for each item of a list
     (list items read through `useListItem` or a selector), and a fetch for each
     keystroke without a debounce.
   - Each `useEffect` must sync with a system outside React: a subscription, the DOM, or
     a timer. An effect that fetches, derives a value, or tells the parent about a
     change is a finding. Put that work in render, in a Flux query, or in the event
     handler.
   - Every effect cleans up. It unsubscribes, clears timers, and aborts or ignores a
     fetch that is no longer current. An effect that only has to be mounted follows the
     Console `SideEffect` rule in `console/CLAUDE.md`.
7. **Duplicated state.** Does each piece of state have one owner?
   - Flux, Redux, form state, and local `useState` each own different data. A value
     copied from one of them into another needs code to keep the two copies the same.
     That code is the defect, even when it works.
   - Look for the signs: a `useEffect` that copies a prop, a query answer, or a store
     value into state; two setters called together; a local copy of Core data that a
     Flux listener already keeps current; one value kept in Redux and in a component at
     the same time.
   - The fix is to read from the owner and derive the rest during render. A draft that
     must change apart from its source, such as a form before save, is the only valid
     copy. It must have a clear point where it starts again from the source.
