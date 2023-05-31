export type RenderFunction = () => Promise<void>;

export class RenderQueue {
  queue: Record<string, RenderFunction>;
  requested: boolean = false;

  constructor() {
    this.queue = {};
    setInterval(() => {
      if (Object.keys(this.queue).length === 0) return;
      void this.render();
    }, 1000 / 60);
  }

  push(key: string, render: RenderFunction): void {
    this.queue[key] = render;
  }

  async render(): Promise<void> {
    const queue = this.queue;
    this.queue = {};
    const keys = Object.keys(queue);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      await queue[key]();
    }
    this.requested = false;
  }
}
