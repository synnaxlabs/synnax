CREATE TABLE "referral" (
	"key" text PRIMARY KEY NOT NULL,
	"recipient" text NOT NULL,
	"created_by" text NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"opened_at" timestamp with time zone,
	"user_id" text,
	"claimed_at" timestamp with time zone,
	"mailed_at" timestamp with time zone,
	CONSTRAINT "referral_user_id_unique" UNIQUE("user_id")
);
