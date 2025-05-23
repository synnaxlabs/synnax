import { Mutex as Core } from "async-mutex";

export type Mutex<G> = G & Core;

export class mutex<G> extends Core {
  static new<G>(guard: G): Mutex<G> & G {
    return new mutex(guard) as Mutex<G> & G;
  }

  constructor(guard: G) {
    super();
    Object.assign(this, guard);
  }
}

export const newMutex = <G>(guard: G): Mutex<G> => new mutex(guard) as Mutex<G>;
