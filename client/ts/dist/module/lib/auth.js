import { z } from 'zod';
import { AuthError } from './errors';
import { UserPayloadSchema } from './user/payload';
export const tokenMiddleware = (token) => {
    return async (md, next) => {
        try {
            md.params['Authorization'] = `Bearer ${await token()}`;
        }
        catch (err) {
            return err;
        }
        return await next(md);
    };
};
export const InsecureCredentialsSchema = z.object({
    username: z.string(),
    password: z.string(),
});
export const TokenResponseSchema = z.object({
    token: z.string(),
    user: UserPayloadSchema,
});
export default class AuthenticationClient {
    static ENDPOINT = '/auth/login';
    token;
    client;
    credentials;
    authenticating;
    authenticated;
    user;
    constructor(factory, creds) {
        this.client = factory.postClient();
        this.credentials = creds;
        this.authenticated = false;
        this.authenticate();
    }
    authenticate() {
        this.authenticating = new Promise((resolve, reject) => {
            this.client
                .send(AuthenticationClient.ENDPOINT, this.credentials, TokenResponseSchema)
                .then(([res, err]) => {
                if (err) {
                    reject(err);
                    return;
                }
                this.token = res?.token;
                this.user = res?.user;
                this.authenticated = true;
                resolve();
            });
        });
    }
    async maybeWaitAuthenticated() {
        if (this.authenticating)
            await this.authenticating;
        this.authenticating = undefined;
    }
    middleware() {
        return tokenMiddleware(async () => {
            await this.maybeWaitAuthenticated();
            if (!this.token) {
                throw new AuthError('[auth] - attempting to authenticate without a token');
            }
            return this.token;
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYXV0aC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uL3NyYy9saWIvYXV0aC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFLQSxPQUFPLEVBQUUsQ0FBQyxFQUFFLE1BQU0sS0FBSyxDQUFDO0FBRXhCLE9BQU8sRUFBRSxTQUFTLEVBQUUsTUFBTSxVQUFVLENBQUM7QUFDckMsT0FBTyxFQUFlLGlCQUFpQixFQUFFLE1BQU0sZ0JBQWdCLENBQUM7QUFFaEUsTUFBTSxDQUFDLE1BQU0sZUFBZSxHQUFHLENBQUMsS0FBNEIsRUFBYyxFQUFFO0lBQzFFLE9BQU8sS0FBSyxFQUFFLEVBQUUsRUFBRSxJQUFJLEVBQUUsRUFBRTtRQUN4QixJQUFJO1lBQ0YsRUFBRSxDQUFDLE1BQU0sQ0FBQyxlQUFlLENBQUMsR0FBRyxVQUFVLE1BQU0sS0FBSyxFQUFFLEVBQUUsQ0FBQztTQUN4RDtRQUFDLE9BQU8sR0FBRyxFQUFFO1lBQ1osT0FBTyxHQUFZLENBQUM7U0FDckI7UUFDRCxPQUFPLE1BQU0sSUFBSSxDQUFDLEVBQUUsQ0FBQyxDQUFDO0lBQ3hCLENBQUMsQ0FBQztBQUNKLENBQUMsQ0FBQztBQUVGLE1BQU0sQ0FBQyxNQUFNLHlCQUF5QixHQUFHLENBQUMsQ0FBQyxNQUFNLENBQUM7SUFDaEQsUUFBUSxFQUFFLENBQUMsQ0FBQyxNQUFNLEVBQUU7SUFDcEIsUUFBUSxFQUFFLENBQUMsQ0FBQyxNQUFNLEVBQUU7Q0FDckIsQ0FBQyxDQUFDO0FBR0gsTUFBTSxDQUFDLE1BQU0sbUJBQW1CLEdBQUcsQ0FBQyxDQUFDLE1BQU0sQ0FBQztJQUMxQyxLQUFLLEVBQUUsQ0FBQyxDQUFDLE1BQU0sRUFBRTtJQUNqQixJQUFJLEVBQUUsaUJBQWlCO0NBQ3hCLENBQUMsQ0FBQztBQUlILE1BQU0sQ0FBQyxPQUFPLE9BQU8sb0JBQW9CO0lBQy9CLE1BQU0sQ0FBQyxRQUFRLEdBQUcsYUFBYSxDQUFDO0lBQ2hDLEtBQUssQ0FBcUI7SUFDMUIsTUFBTSxDQUFjO0lBQ3BCLFdBQVcsQ0FBc0I7SUFDekMsY0FBYyxDQUE0QjtJQUMxQyxhQUFhLENBQVU7SUFDdkIsSUFBSSxDQUEwQjtJQUU5QixZQUFZLE9BQTBCLEVBQUUsS0FBMEI7UUFDaEUsSUFBSSxDQUFDLE1BQU0sR0FBRyxPQUFPLENBQUMsVUFBVSxFQUFFLENBQUM7UUFDbkMsSUFBSSxDQUFDLFdBQVcsR0FBRyxLQUFLLENBQUM7UUFDekIsSUFBSSxDQUFDLGFBQWEsR0FBRyxLQUFLLENBQUM7UUFDM0IsSUFBSSxDQUFDLFlBQVksRUFBRSxDQUFDO0lBQ3RCLENBQUM7SUFFRCxZQUFZO1FBQ1YsSUFBSSxDQUFDLGNBQWMsR0FBRyxJQUFJLE9BQU8sQ0FBQyxDQUFDLE9BQU8sRUFBRSxNQUFNLEVBQUUsRUFBRTtZQUNwRCxJQUFJLENBQUMsTUFBTTtpQkFDUixJQUFJLENBQ0gsb0JBQW9CLENBQUMsUUFBUSxFQUM3QixJQUFJLENBQUMsV0FBVyxFQUNoQixtQkFBbUIsQ0FDcEI7aUJBQ0EsSUFBSSxDQUFDLENBQUMsQ0FBQyxHQUFHLEVBQUUsR0FBRyxDQUFDLEVBQUUsRUFBRTtnQkFDbkIsSUFBSSxHQUFHLEVBQUU7b0JBQ1AsTUFBTSxDQUFDLEdBQUcsQ0FBQyxDQUFDO29CQUNaLE9BQU87aUJBQ1I7Z0JBQ0QsSUFBSSxDQUFDLEtBQUssR0FBRyxHQUFHLEVBQUUsS0FBSyxDQUFDO2dCQUN4QixJQUFJLENBQUMsSUFBSSxHQUFHLEdBQUcsRUFBRSxJQUFJLENBQUM7Z0JBQ3RCLElBQUksQ0FBQyxhQUFhLEdBQUcsSUFBSSxDQUFDO2dCQUMxQixPQUFPLEVBQUUsQ0FBQztZQUNaLENBQUMsQ0FBQyxDQUFDO1FBQ1AsQ0FBQyxDQUFDLENBQUM7SUFDTCxDQUFDO0lBRU8sS0FBSyxDQUFDLHNCQUFzQjtRQUNsQyxJQUFJLElBQUksQ0FBQyxjQUFjO1lBQUUsTUFBTSxJQUFJLENBQUMsY0FBYyxDQUFDO1FBQ25ELElBQUksQ0FBQyxjQUFjLEdBQUcsU0FBUyxDQUFDO0lBQ2xDLENBQUM7SUFFRCxVQUFVO1FBQ1IsT0FBTyxlQUFlLENBQUMsS0FBSyxJQUFJLEVBQUU7WUFDaEMsTUFBTSxJQUFJLENBQUMsc0JBQXNCLEVBQUUsQ0FBQztZQUNwQyxJQUFJLENBQUMsSUFBSSxDQUFDLEtBQUssRUFBRTtnQkFDZixNQUFNLElBQUksU0FBUyxDQUNqQixxREFBcUQsQ0FDdEQsQ0FBQzthQUNIO1lBQ0QsT0FBTyxJQUFJLENBQUMsS0FBSyxDQUFDO1FBQ3BCLENBQUMsQ0FBQyxDQUFDO0lBQ0wsQ0FBQyJ9