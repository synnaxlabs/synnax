import { Synnax, Connectivity } from "@synnaxlabs/client";

class ClientPool {
  private clients = new Map<string, Synnax>();

  acquire(host: string): Synnax {
    const c = this.clients.get(host);
    if (!c) throw new Error("No client for target: " + host);
    return c;
  }

  register(host: string, c: Synnax) {
    if (c.connectivity.status() != Connectivity.CONNECTED) {
      console.warn(
        "Registering client with unhealthy connection: " +
          c.connectivity.error()
      );
    }
    this.clients.set(host, c);
  }
}

const clientPool = new ClientPool();

export default clientPool;
