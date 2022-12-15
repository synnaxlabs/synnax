import { Middleware } from "../middleware";

export const logMiddleware = (): Middleware => {
  return async (md, next) => {
    console.log(JSON.stringify(md, undefined, 2));
    const err = await next(md);
    if (err != null) {
      console.log(err);
    }
    return err;
  };
};
