ALTER TABLE "activation" ADD COLUMN "name" text;--> statement-breakpoint
ALTER TABLE "activation" ADD COLUMN "renewal_secret_hash" text;--> statement-breakpoint
ALTER TABLE "activation" ADD CONSTRAINT "activation_renewal_secret_hash_unique" UNIQUE("renewal_secret_hash");