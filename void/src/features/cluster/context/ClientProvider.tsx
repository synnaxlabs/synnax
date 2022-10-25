import { Synnax, Connectivity } from "@synnaxlabs/client";

class ClientPool {
  private clients = new Map<string, Synnax>();
  private singleton = false;

  constructor(singleton = false) {
    this.singleton = singleton;
  }

  acquire(key: string): Synnax | undefined {
    return this.clients.get(key);
  }

  set(key: string, c: Synnax) {
    if (this.singleton) this.closeAll();
    this.clients.set(key, c);
  }

  closeAll() {
    this.clients.forEach((c) => c.close());
    this.clients.clear();
  }
}

const clientPool = new ClientPool(/* singleton */ true);

export default clientPool;
