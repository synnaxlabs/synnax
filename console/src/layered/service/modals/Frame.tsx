import { Dialog } from "@synnaxlabs/pluto";

export interface FrameProps extends Dialog.DialogProps {}

export const Frame = (props: FrameProps) => <Dialog.Dialog full {...props} />;
