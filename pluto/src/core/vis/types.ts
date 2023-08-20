import { AetherComponent } from "../../aether/aether";

export interface Renderable<P> extends AetherComponent {
  render: (props: P) => Promise<void>;
}
