import { TelemSourceMeta } from "./TelemSource";

export interface TelemProvider {
  get: <T extends TelemSourceMeta>(key: string) => T;
}
