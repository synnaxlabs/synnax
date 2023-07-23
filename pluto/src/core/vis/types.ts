import { AetherComponent } from "../aether/worker";

export interface Renderable<P> extends AetherComponent {
  render: (props: P) => Promise<void>;
}
