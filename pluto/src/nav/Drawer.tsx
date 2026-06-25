import { type box } from "@synnaxlabs/x";

import { type Resize } from "@/resize";

export interface DrawerProps extends Resize.SingleProps {
  collapsed?: boolean;
  collapseThreshold?: number;
  onCollapse?: () => void;
}

export const Drawer = ({
  onCollapse,
  collapsed,
  collapseThreshold,
  ...rest
}: DrawerProps) => {
  const handleDrag = (region: box.Box) => {};
  return <Resize.Single {...rest} />;
};
