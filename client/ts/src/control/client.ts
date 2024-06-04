import { StateTracker } from "@/control/state";
import { framer } from "@/framer";

const CONTROL_STATE_KEY = "sy_node_1_control";

export class Client {
  private readonly framer: framer.Client;

  constructor(framer: framer.Client) {
    this.framer = framer;
  }

  async openStateTracker(): Promise<StateTracker> {
    const stream = await this.framer.openStreamer(CONTROL_STATE_KEY);
    return new StateTracker(stream);
  }
}
