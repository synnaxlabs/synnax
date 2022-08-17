package delta

type Client struct {
	config Config
}

func New(cfg Config) Client {
	return Client{config: cfg}
}
