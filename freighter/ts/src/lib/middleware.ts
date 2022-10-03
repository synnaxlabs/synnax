export type MD = {
  target: string;
  protocol: string;
  params?: Record<string, string>;
};

export type Next = (md: MD) => Promise<Error | undefined>;

export type Middleware = (md: MD, next: Next) => Promise<Error | undefined>;

type Finalizer = (md: MD) => Promise<Error | undefined>;

export const runSequentially = (
  md: MD,
  middleware: Middleware[],
  finalizer: Finalizer
): Promise<Error | undefined> => {
  let i = 0;
  const next = (md: MD): Promise<Error | undefined> => {
    if (i == middleware.length) return finalizer(md);
    const _mw = middleware[i];
    i++;
    return _mw(md, next);
  };
  return next(md);
};

export class MiddlewareCollector {
  middleware: Middleware[] = [];

  use(...mw: Middleware[]) {
    this.middleware.push(...mw);
  }

  executeMiddleware(md: MD, finalizer: Finalizer): Promise<Error | undefined> {
    return runSequentially(md, this.middleware, finalizer);
  }
}
