import { Tabs } from "@synnaxlabs/pluto";
import { SiTypescript, SiPython } from "react-icons/si";

const TABS = [
  {
    tabKey: "python",
    name: "Python",
    icon: <SiPython />,
  },
  {
    tabKey: "typescript",
    name: "Typescript",
    icon: <SiTypescript />,
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
