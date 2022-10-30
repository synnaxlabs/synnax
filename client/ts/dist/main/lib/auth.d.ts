import { HTTPClientFactory, Middleware } from '@synnaxlabs/freighter';
import { z } from 'zod';
import { UserPayload } from './user/payload';
export declare const tokenMiddleware: (token: () => Promise<string>) => Middleware;
export declare const InsecureCredentialsSchema: z.ZodObject<{
    username: z.ZodString;
    password: z.ZodString;
}, "strip", z.ZodTypeAny, {
    username: string;
    password: string;
}, {
    username: string;
    password: string;
}>;
export declare type InsecureCredentials = z.infer<typeof InsecureCredentialsSchema>;
export declare const TokenResponseSchema: z.ZodObject<{
    token: z.ZodString;
    user: z.ZodObject<{
        key: z.ZodString;
        username: z.ZodString;
    }, "strip", z.ZodTypeAny, {
        key: string;
        username: string;
    }, {
        key: string;
        username: string;
    }>;
}, "strip", z.ZodTypeAny, {
    token: string;
    user: {
        key: string;
        username: string;
    };
}, {
    token: string;
    user: {
        key: string;
        username: string;
    };
}>;
export declare type TokenResponse = z.infer<typeof TokenResponseSchema>;
export default class AuthenticationClient {
    private static ENDPOINT;
    private token;
    private client;
    private credentials;
    authenticating: Promise<void> | undefined;
    authenticated: boolean;
    user: UserPayload | undefined;
    constructor(factory: HTTPClientFactory, creds: InsecureCredentials);
    authenticate(): void;
    private maybeWaitAuthenticated;
    middleware(): Middleware;
}
