CREATE TABLE "invitation" (
	"key" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization" uuid NOT NULL,
	"email" text NOT NULL,
	"role" text NOT NULL,
	"sent_at" timestamp with time zone DEFAULT now() NOT NULL,
	"expires_at" timestamp with time zone NOT NULL,
	CONSTRAINT "invitation_organization_email_unique" UNIQUE("organization","email")
);
--> statement-breakpoint
ALTER TABLE "invitation" ADD CONSTRAINT "invitation_organization_organization_key_fk" FOREIGN KEY ("organization") REFERENCES "public"."organization"("key") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "invitation_email_idx" ON "invitation" USING btree ("email");