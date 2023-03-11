import { Icon } from "@synnaxlabs/media";
import { Tabs } from "@synnaxlabs/pluto";

const TABS = [
  {
    tabKey: "python",
    name: "Python",
    icon: <Icon.Python />,
  },
  {
    tabKey: "typescript",
    name: "Typescript",
    icon: <Icon.Typescript />,
  },
];

export const ClientTabs = (props: any): JSX.Element => {
  const tabsProps = Tabs.useStatic({ tabs: TABS });
  return (
    <Tabs {...tabsProps}>
      {(tab) => (
        <div>
          <h2>Using {tab.name}</h2>
          {props[`setup-${tab.tabKey}`]}
          {props[tab.tabKey]}
        </div>
      )}
    </Tabs>
  );
};
