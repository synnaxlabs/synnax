import { Icon } from "@synnaxlabs/media";
import { Tabs } from "@synnaxlabs/pluto";

const TABS = [
  {
    tabKey: "npm",
    name: "npm",
    icon: <Icon.NPM />,
  },
  {
    tabKey: "yarn",
    name: "yarn",
    icon: <Icon.Yarn />,
  },
  {
    tabKey: "pnpm",
    name: "pnpm",
    icon: <Icon.PNPM />,
  },
];

export interface PackageManagerTabsProps {
  yarn?: string;
  npm?: string;
  pnpm?: string;
}

export const PackageManagerTabs = (props: PackageManagerTabsProps): JSX.Element => {
  const tabs = TABS.filter(({ tabKey }) => tabKey in props).map(
    ({ tabKey, name, icon }) => ({
      tabKey,
      name,
      icon,
    })
  );
  const tabsProps = Tabs.useStatic({ tabs });
  return <Tabs {...tabsProps}>{(tab) => props[tab.tabKey]}</Tabs>;
};
