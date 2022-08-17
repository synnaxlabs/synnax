import {Authentication, LoginRequest} from "./auth/auth";
import { User } from "./user/user";
import {HTTPClient} from "./http/client";

type Options = {
  endpoint: string;
  port: string;
  username: string;
  password: string;
  useSSL: boolean;
};

class Client {
  private options: Options;
  private auth: Authentication;
  constructor(options: Options) {
    this.options = options;

    let client = new HTTPClient(host: );

    /* Only authenticate if a username or password are provided,
      otherwise assume the user will call Authenticate at an opportune time. */
    if (this.options.username === "" && this.options.password === "") {
      this.authenticate()
    }
  }

  async Authenticate(req: LoginRequest) {
    this.options.username = req.username;
    this.options.password = req.password;
    await this.auth.login(req);
  }

  async authenticate() {
    await this.auth.login({
      username: this.options.username,
      password: this.options.password,
    });
  }

  User(): User {
    return this.auth.user;
  }
}

const main = () => {
  const options: Options = {
    endpoint: "http://localhost:8080",
    port: "8080",
    username: "admin",
    password: "admin",
    useSSL: false,
  };
  const client = new Client(options);
};
