import { Tab, Tabs as PTabs } from "@synnaxlabs/pluto";

export type TabsProps = {
  tabs: Tab[];
} & Record<string, JSX.Element | any>;

export const Tabs = ({ tabs, ...props }: TabsProps): JSX.Element => {
  tabs = tabs.map((tab) => ({ ...tab, icon: props[`${tab.tabKey}-icon`] }));
  const staticProps = PTabs.useStatic({ tabs });

  return <PTabs {...staticProps}>{(tab) => <div>{props[tab.tabKey]}</div>}</PTabs>;
};
