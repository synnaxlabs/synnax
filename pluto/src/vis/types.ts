import { aether } from "@/aether/aether";

export interface Renderable<P> extends aether.Component {
  render: (props: P) => Promise<void>;
}
