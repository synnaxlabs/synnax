
export const tokenMiddleware = (token: string): Middleware => {
    return async (md: MD, next: (md: MD) => Promise<Error | undefined>) => {
        md['Authorization'] = `Bearer ${token}`
        return next(md)
    }
}


export const logMiddleware = (): Middleware => {
    return async (md: MD, next: (md: MD) => Promise<Error | undefined>) => {
        console.log(md);
        const err = await next(md);
        if (err) {
            console.log(err)
        }
        return err
    }
}

