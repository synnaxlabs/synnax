import { Resize } from "../resize";
export interface DrawerProps extends Resize.SingleProps {
    collapsed?: boolean;
    collapseThreshold?: number;
    onCollapse?: () => void;
}
export declare const Drawer: ({ onCollapse, collapsed, collapseThreshold, onResize, onResizeEnd, className, ...rest }: DrawerProps) => import("react").JSX.Element;
//# sourceMappingURL=Drawer.d.ts.map