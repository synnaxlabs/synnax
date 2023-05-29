export type RenderFunction = () => Promise<void>;

export class RenderQueue {
  queue: RenderFunction[];
  requested: boolean = false;

  constructor() {
    this.queue = [];
  }

  push(render: RenderFunction): void {
    this.queue.push(render);
    if (!this.requested) {
      this.requested = true;
      requestAnimationFrame(() => {
        void this.render();
      });
    }
  }

  async render(): Promise<void> {
    this.requested = false;
    const toRender = this.queue;
    this.queue = [];
    for (const render of toRender) await render();
  }
}
