CREATE TABLE "activation" (
	"key" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"license" uuid NOT NULL,
	"fingerprint" text[] NOT NULL,
	"first_seen" timestamp with time zone DEFAULT now() NOT NULL,
	"last_seen" timestamp with time zone DEFAULT now() NOT NULL,
	"released_at" timestamp with time zone
);
--> statement-breakpoint
CREATE TABLE "event" (
	"key" serial PRIMARY KEY NOT NULL,
	"at" timestamp with time zone DEFAULT now() NOT NULL,
	"kind" text NOT NULL,
	"actor" text NOT NULL,
	"organization" uuid,
	"license" uuid,
	"activation" uuid,
	"detail" jsonb DEFAULT '{}'::jsonb NOT NULL
);
--> statement-breakpoint
CREATE TABLE "license" (
	"key" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization" uuid NOT NULL,
	"edition" text NOT NULL,
	"term" text NOT NULL,
	"nodes" integer NOT NULL,
	"channels" integer DEFAULT 0 NOT NULL,
	"expires_at" timestamp with time zone,
	"max_version" text,
	"label" text DEFAULT '' NOT NULL,
	"issued_by" text NOT NULL,
	"issued_at" timestamp with time zone DEFAULT now() NOT NULL,
	"revoked_at" timestamp with time zone
);
--> statement-breakpoint
CREATE TABLE "organization" (
	"key" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"kind" text NOT NULL,
	"name" text NOT NULL,
	"clerk_org_id" text,
	"plain_tenant_id" text,
	"owner_user_id" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "organization_clerk_org_id_unique" UNIQUE("clerk_org_id"),
	CONSTRAINT "organization_owner_user_id_unique" UNIQUE("owner_user_id")
);
--> statement-breakpoint
ALTER TABLE "activation" ADD CONSTRAINT "activation_license_license_key_fk" FOREIGN KEY ("license") REFERENCES "public"."license"("key") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "event" ADD CONSTRAINT "event_organization_organization_key_fk" FOREIGN KEY ("organization") REFERENCES "public"."organization"("key") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "event" ADD CONSTRAINT "event_license_license_key_fk" FOREIGN KEY ("license") REFERENCES "public"."license"("key") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "event" ADD CONSTRAINT "event_activation_activation_key_fk" FOREIGN KEY ("activation") REFERENCES "public"."activation"("key") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "license" ADD CONSTRAINT "license_organization_organization_key_fk" FOREIGN KEY ("organization") REFERENCES "public"."organization"("key") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "activation_license_idx" ON "activation" USING btree ("license");--> statement-breakpoint
CREATE INDEX "event_license_idx" ON "event" USING btree ("license");--> statement-breakpoint
CREATE INDEX "event_actor_at_idx" ON "event" USING btree ("actor","at");--> statement-breakpoint
CREATE INDEX "event_organization_at_idx" ON "event" USING btree ("organization","at");--> statement-breakpoint
CREATE INDEX "license_organization_idx" ON "license" USING btree ("organization");