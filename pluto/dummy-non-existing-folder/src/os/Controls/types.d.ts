import { type Flex } from "@synnaxlabs/lyra/flex";
import { type runtime } from "@synnaxlabs/x";
export type ControlsAction = "close" | "minimize" | "maximize";
export interface InternalControlsProps extends Flex.BoxProps {
    forceOS?: runtime.OS;
    disabled?: ControlsAction[];
    focused?: boolean;
    maximized?: boolean;
    onMinimize?: () => void;
    onMaximize?: () => void;
    onFullscreen?: () => void;
    onClose?: () => void;
}
//# sourceMappingURL=types.d.ts.map