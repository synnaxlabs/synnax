import type { MiddlewareHandler } from "astro";

export const onRequest: MiddlewareHandler = async (context, next) => {
  const response = await next();

  response.headers.set(
    "Content-Security-Policy",
    "default-src 'self'; script-src 'self' 'unsafe-inline' https://vercel.live https://us-assets.i.posthog.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; connect-src 'self' https://ywd9t0jxcs-dsn.algolia.net https://raw.githubusercontent.com https://us.i.posthog.com; img-src 'self' data: https://us-assets.i.posthog.com; frame-src https://vercel.live; object-src 'none';",
  );

  return response;
};
