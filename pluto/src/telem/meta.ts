import { TelemSourceMeta } from "@/core/vis/telem";

export interface ModifiableTelemSourceMeta extends TelemSourceMeta {
  setProps: (props: any) => void;
  cleanup: () => void;
}
