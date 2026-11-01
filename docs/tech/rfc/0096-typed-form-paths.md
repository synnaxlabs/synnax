# 96 Typed form paths

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-01
- **Related**: [RFC 0065 - Pluto form optionality](0065-pluto-form-optionality.md),
  [RFC 0084 - Collaborative forms](0084-collaborative-forms.md)

## 0 Summary

A form field path is a plain string. A typo, a string sent to a numeric input, and a
field that the current channel type lacks all compile, then fail at run time.

This RFC makes a path a typed value that reads like the data it points at. The type
checker, completion, go-to-definition, and rename all follow it to the schema:

```tsx
const path = deep.path<Values>();

<Form.TextField path={path.name} />
<Form.NumericField path={path.config.channels[key].port} />
```

A reusable component takes a path as a prop. The prop states the shape it needs, and any
form that holds that shape can host it:

```tsx
const ScaleFields = ({ path }: { path: deep.Path<LinearScale> }) => (
  <>
    <Form.NumericField path={path.slope} />
    <Form.NumericField path={path.yIntercept} />
  </>
);
```

Plain strings stop compiling at the end of the migration. Behind the form API a path
stays a string, so no action, wire format, or stored shape changes.

## 1 Motivation

Form code outside specs holds about 1,120 path arguments: 855 in the Console, 218 in
Pluto, 40 in the sites, and 4 in Lyra.

Of these, about 510 are literals, 332 are forwarded props or variables, and 240 are
templates.

The types around these paths are claims that nothing checks:

- **Explicit type arguments**: 360 sites such as `useFieldValue<string>(path)`.
- **`fieldKey`**: 107 literals hide the last part of a path from the reader and the
  compiler.
- **Parent lookups**: 9 `onChange` handlers cut the string with
  `path.slice(0, path.lastIndexOf("."))`.
- **`ContextValue<Z>`**: Only `reset` and `value` use `Z`. `set` and `get` take strings
  (`lyra/src/form/Context.ts`).

The usual fix is a union of every valid path string. It now compiles: on TypeScript 7
the old depth failure does not reproduce on the task forms or on the 98-member
`ElementConfig` union. It still cannot type the 51% of sites that a prefix or a
forwarded prop builds, or a path through `config` in code generic over `task.Schemas`.

## 2 Principles

1. **A path has exactly the fields its value has**: The same rule as TypeScript on a
   plain value. A field that only some union members hold needs a narrow first. Reading
   it from the wrong member throws at run time today, even for an optional field, unless
   the caller passes `optional: true`.
2. **Types stop at the form API**: Lyra converts a path to a string once at each entry.
   The store, statuses, actions, and the wire keep strings. So does the `path` that
   `Form.use` reports to `onChange`, because it is an output.
3. **Run-time checks stay**: Values come from people and from the Core. The schema
   validates every write, and an unknown path still throws (RFC 0065).

## 3 Design

### 3.0 The `deep.Path` type

```ts
declare const VALUE: unique symbol;

export interface Target<V> {
  readonly [VALUE]: V;
  [Symbol.toPrimitive](this: void): string;
}

type Primitive = string | number | boolean | bigint | symbol | null | undefined;

export type Path<V> = Target<V> & Children<V>;

interface Open {
  readonly [key: string]: Path<unknown>;
}

type Children<V> = unknown extends V
  ? Open
  : [V] extends [Primitive]
    ? unknown
    : [V] extends [NonNullable<V>]
      ? [V] extends [readonly (infer E)[]]
        ? { readonly [key: string | number]: Path<E> }
        : { readonly [K in keyof V & string]-?: Path<V[K]> }
      : unknown;
```

- **`deep.Path<V>`**: What a caller holds. Its properties are the fields of `V`.
- **`deep.Target<V>`**: What an API accepts. It is covariant, so `Path<AIVoltage>` flows
  into a prop typed `Path<{ minVal: number; maxVal: number }>` with no cast.

TypeScript evaluates a mapped property only when a caller reads it, so large unions cost
little and recursive shapes stay lazy.

### 3.1 Run time

```ts
const node = (path: string, nodes: Map<string, unknown>): unknown =>
  new Proxy(
    {},
    {
      get: (_, key) => {
        if (key === Symbol.toPrimitive) return () => path;
        if (typeof key === "symbol") return undefined;
        const child = path === "" ? key : `${path}.${key}`;
        if (!nodes.has(child)) nodes.set(child, node(child, nodes));
        return nodes.get(child);
      },
    },
  );

export const path = <V>(): Path<V> => node("", new Map()) as Path<V>;
```

- **String form**: `String(path)` gives the dot-separated path. A `state.usePersisted`
  key is not a schema field, so it stays a template over `String(path)`.
- **Stable identity**: A root caches each child, so `path.slope === path.slope`. The
  React Compiler memoizes on that comparison. No caller can observe the cache, the same
  as `hashes` at `client/ts/src/query/query.ts:31`.
- **Field names**: The whole API is a symbol key or a free function, so a schema field
  can have any name. A `then` field reads as an object, so `await` stays safe.
- **Cost**: 30 to 80 ns per path, and about 1 KB per keyed list entry that a form reads.
  1,000 paths in one render add about 0.08 ms.

### 3.2 Narrowing

| Value                 | Fields without a narrow | Narrow                                     |
| --------------------- | ----------------------- | ------------------------------------------ |
| Union                 | Those on every member   | `deep.variant(path, "type", "ai_voltage")` |
| Optional              | None                    | `deep.present(path)`                       |
| Generic `C extends T` | None                    | `deep.widen<T>(path)`                      |

Each function returns the same object and changes only its type. `variant` also takes a
union of values, and a type parameter in a correlated dispatch.

A component that reads a field only some members hold takes `deep.Target<Shape>` with
that field optional, then reads through `deep.widen(path)`. `useChannelDevice`,
`useAnalogInputPort`, and `CustomScaleSection` work this way.

### 3.3 Form API

| Surface                                      | Today                            | New                                              |
| -------------------------------------------- | -------------------------------- | ------------------------------------------------ |
| `useField`, `useFieldValue`, `useFieldState` | `<I>(path: string)`              | `<I>(path: deep.Target<I>)`                      |
| `useFieldList`, `useFieldListUtils`          | `<K, E>(path: string)`           | `<E>(path: deep.Target<E[]>)`                    |
| `get`                                        | `<V>(path: string)`              | `<V>(path: deep.Target<V>)`                      |
| `set`                                        | `(path: string, value: unknown)` | `<V>(path: deep.Target<V>, value: NoInfer<V>)`   |
| `insert`                                     | `<E>(path: string, value: E)`    | `<E>(path: deep.Target<E[]>, value: NoInfer<E>)` |
| `Form.Field`                                 | `path: string`                   | `path: deep.Target<I>`                           |
| Built fields                                 | `path: string`                   | `path: deep.Target<I \| undefined>`              |
| Every other path argument                    | `path: string`                   | `path: deep.Target<unknown>`                     |

A built field accepts an optional value because inputs render unset (RFC 0065).
`optional: true` stays on the hooks for dynamically keyed paths.

### 3.4 Roots and reusable components

A module that owns a form declares one root: `const path = deep.path<Values>()`. A task
form takes its values from `Task.Values<S>`. A component that several forms host takes
`path: deep.Path<Shape>`, which replaces the `path` and `prefix` strings in 51 files.

The task list views, `Task.BindChannels`, and `Task.Views.GroupTree` take the list as a
required `path`, so each form names its list and the entry type follows from it.

### 3.5 Fields that write siblings

The 9 parent lookups exist because `onChange` receives a string. Each one becomes a
component that takes the parent path, so `extra.path` leaves `UseFieldOptions`:

```tsx
const TypeField = ({ path }: { path: deep.Path<AIChannel> }) => (
  <SelectType path={path.type} onChange={(type, { set }) => set(path, ZERO[type])} />
);
```

Five of the 9 sit in `fieldBuilder` calls at module scope, and those calls become
components.

### 3.6 Deleted code

- **`fieldKey`**: On `Form.Field`, `fieldBuilder`, and `CheckboxField`.
- **Explicit type arguments**: The 360 sites, because a path now states its value.
- **Parent lookups**: The 9 sites and `extra.path`.
- **`deep.Key`** (`x/ts/src/deep/path.ts:43`) **and `deep.Join`**
  (`x/ts/src/deep/join.ts:10`): No caller uses them.

### 3.7 Evidence

All results are from TypeScript 7.0.2 on the NI analog input form: 8 files and 30
channel types.

- **Real forms**: On real proxies, all 14 NI spec files passed (195 tests) against a
  live Core with the React Compiler on. A temporary layer called `String(path)` before
  Lyra. Breaking the proxy join failed 64 of 67 tests in the two specs that were run.
- **Site changes**: 0 type errors and 29 fewer lines. The rule in §2 fails 3 of about
  190 paths: `useChannelDevice`, `useAnalogInputPort`, and `CustomScaleSection`.
- **Compile cost**: About 2% more instantiations on the form. On `ElementConfig`, a
  narrow and a read cost about 300 instantiations, with no TS2589.
- **Mistakes that fail**: An unknown field, a wrong value type, a wrong literal, a wrong
  list entry, a child of an optional parent, a field on only some union members, a shape
  a reusable component does not accept, and a plain string.
- **Editor**: Completion lists only schema fields. Go-to-definition lands on the Zod
  key, and rename and references work through list entries.
- **Full migration**: 0 type errors in every package. All `x`, Lyra, Pluto, Console, and
  site specs that touch forms passed, on a live Core where a spec needs one.

### 3.8 Limits

- **A root is a claim**: React context is untyped, so the compiler cannot check
  `deep.path<V>()` against the enclosing form. A wrong root throws on the first render,
  as a wrong string does today. One claim per module replaces 360.
- **List keys are open**: `path.channels[key]` takes any string, so
  `path.channels.lenght` compiles. The fields under it stay typed.
- **Writes are covariant**: A component typed `Path<{ mode: string }>` can write a value
  that the real union rejects. The schema catches it at run time.
- **Correlated writes**: A write that spreads over a union member loses the link between
  `type` and the other fields, so the 8 channel and scale type changes keep an `as`.
- **Unknown values**: A `z.unknown()` field has no value type. The HTTP static field
  casts its path to the type of the input it renders.

## 4 Implementation phases

Phase 1 adds the types and makes the form API accept strings and paths. Phase 2 moves
every form and spec, then removes the string overload and `fieldKey`. A codemod does
most of Phase 2. The 9 sites from §3.5 and the 3 from §3.7 are hand-written. No flag is
needed, because users see no change.

No wire, stored, or file format changes. Lyra and Pluto are private packages, so the API
change reaches no outside caller. `deep.Key` and `deep.Join` leave `@synnaxlabs/x`.

| Phase | Scope                                             | Touches                                            | Depends on |
| ----- | ------------------------------------------------- | -------------------------------------------------- | ---------- |
| 1     | Accept typed paths in the Lyra form API           | `x/ts/src/deep`, `lyra/src/form`, `pluto/src/flux` | -          |
| 2     | Move every form to typed paths and refuse strings | `console/src`, `pluto/src`, `lyra/src`, `site`     | 1          |

## 5 Resolved decisions

1. **A union of path strings**: The choice of React Hook Form, TanStack Form, and
   Superforms. Rejected because it leaves prefixes and forwarded props to casts, which
   are 51% of the sites here. Those libraries leave reusable components the same way.
2. **A branded string with `deep.at`**: `deep.at(prefix, "slope")` catches the same
   mistakes at 1.8% with no `Proxy`, as Conform's `FieldName` does. Rejected because
   every path gains a function call, and a string gets completion but no
   go-to-definition or rename. The trade is real: `path.slope` is 10 to 30 times slower
   per path and is the first production `Proxy` in the TypeScript packages. Angular's
   `FieldTree` chose the same property form.
3. **Plain templates over a brand**: `` `${path}.slope` `` changes the least code.
   Rejected because the value type does not read back out, and a template cannot pass
   into a typed prop.
4. **Keep plain strings compiling**: An optional brand lets migration stop anywhere.
   Rejected because a typo stays legal, so the check holds only where a caller opts in.
   The trade is real: the literals in forms and about 600 in specs need a codemod.
5. **Loose unions**: Reading any member's field as `V | undefined` changes no call site.
   Rejected because a read can still throw at run time, and it needs three more helper
   types.
6. **Oracle-generated paths**: Rejected because Console form schemas are written by
   hand.
7. **No cache in a root**: Simpler, and it holds no state. Rejected because each read
   makes a new object, so every memo in the React Compiler output misses.
8. **A bound kit**: `const F = Form.bind(schema)` checks a literal path against one
   schema. Rejected because it adds a second concept and leaves forwarded paths untyped.
9. **A phase for each area**: Keeps every PR near the size target. Rejected because an
   area is not a seam: a typed form next to a string component needs a temporary
   conversion, and the NI analog input run needed 21 of them. The trade is real: Phase 2
   changes about 1,700 sites, so the compiler and the specs carry its review, and open
   PRs that touch form code must convert.
