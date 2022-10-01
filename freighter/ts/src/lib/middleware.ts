export type MD = {
    target: string,
    [key: string]: string
}

export type Middleware = (
    md: MD,
    next: (md: MD) => Promise<Error | undefined>
) =>  Promise<Error | undefined>;

type Finalizer = (md: MD) => Promise<Error | undefined>;

export const runSequentially = (
    md: MD,
    middleware: Middleware[],
    finalizer: Finalizer
): Promise<Error | undefined> => {
    const next = (md: MD) => {
        if (middleware.length === 0) return finalizer(md);
        const [nextMW, ..._mws] = middleware
        middleware = _mws
        return nextMW(md, next);
    }
    return next(md);
}

