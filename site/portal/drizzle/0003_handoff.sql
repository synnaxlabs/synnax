CREATE TABLE "handoff" (
	"code_hash" text PRIMARY KEY NOT NULL,
	"key" text NOT NULL,
	"expires_at" timestamp with time zone NOT NULL
);
