# Useful React Patterns

## Composition

Composition is perhaps the most powerful pattern in software engineering, and, until the
introduction of the Context API, building composite components in React has been
challenging. Pluto's [header](../../pluto/src/core/std/Header) is a good example.
Imagine we have a header with some specified font size. Our simple header component
might look like this:

```jsx
import { Text } from "@/core/std/Typography";

const Header = ({
  children,
  // The font level
  level,
}) => <Text level={level}>{children}</Text>;
```

Now, imagine we want to add some actions to the header, such as a button to create some
new content. The most naive vay of implementing this is to add an optional `actions`
prop to the header, and render it like so:

```jsx
import { Typography, Text } from "@/core/std/Typography";
import { Use } from "@/core/std/Use";

const Header = ({
  children,
  // The font level
  level,
  // Actions to render
  actions,
}) => (
  <div>
    <Text level={level}>{children}</Text>
    {actions.map((action, i) => (
      // Notice how we size the button off of the typography level.
      <Use key={i} size={Typography.componentSizeLevels[level]}>
        {action}
      </Use>
    ))}
  </div>
);
```

This works, but it's not very flexible. What if we want to make it easy to add a styled
tab selector instead? Now we need to add more props to the header. What we end up with
is a component that has a lot of props, and whose API is unintuitive to use.

This is where composition and context comes in handy. Instead of creating one Header
component that does everything, we can create a Header component that serves as a
contextual reference that standardizes the components that are rendered inside of it.

```jsx
import { createContext } from "react";

const HeaderContext = createContext();

const Header = ({ level, children }) => {
  return <HeaderContext.Provider value={{ level }}>{children}</HeaderContext.Provider>;
};
```

Now, we can create a `HeaderTitle` component that will render the title of the header
with the correct font size.

```jsx
const HeaderTitle = ({ children }) => {
  const { level } = useContext(HeaderContext);
  return <Text level={level}>{children}</Text>;
};
```

And a `HeaderActions` component that will render the actions with the correct size.

```jsx
const HeaderActions = ({ children, actions }) => {
  const { level } = useContext(HeaderContext);
  return (
    <div>
      {actions.map((action, i) => (
        <Use
          key={action.key}
          size={Typography.componentSizeLevels[level]}
          onClick={action.onClick}
        >
          {action.title}
        </Use>
      ))}
    </div>
  );
};
```

We can also create a `HeaderTabsSelector` component that will render a tab selector with
the correct size.

```jsx
const HeaderTabsSelector = ({ children, tabs }) => {
  const { level } = useContext(HeaderContext);
  return (
    <div>
      {tabs.map((tab, i) => (
        <Use key={tab.key} size={Typography.componentSizeLevels[level]}>
          {tab.title}
        </Use>
      ))}
    </div>
  );
};
```

Now, we can use our header like so:

```jsx
<Header level={2}>
  <HeaderTitle>My Header</HeaderTitle>
  <HeaderActions
    actions={[
      { key: "new", title: "New" },
      { key: "save", title: "Save" },
    ]}
  />
</Header>
```

If we want to swap out the actions for a tab selector, we can do so easily:

```jsx
<Header level={2}>
  <HeaderTitle>My Header</HeaderTitle>
  <HeaderTabsSelector
    tabs={[
      { key: "tab1", title: "Tab 1" },
      { key: "tab2", title: "Tab 2" },
    ]}
  />
</Header>
```

This API is not only much more flexible, but also much more readable. We make the
interface footprint much less cognitively demanding, and at the same time standardize
the UI presentation.

Composition is a powerful pattern that you'll see used across the Synnax codebase.

## Hooks as Props

## Hooks as Replacements for HOCs
