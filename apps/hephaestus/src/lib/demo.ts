
// Hardcoded configuration to bypass Database
// Paste your values inside the quotes

// Hardcoded keys are deprecated. All credentials are now fetched from the database.
// See src/lib/credentials.ts for the DB interface.

export const DEMO_SSH_KEYS = [];
export const DEMO_SSL_CERTIFICATES = [];

// SSL certificates are now fetched from DB


export const DEMO_ENV_VARS = {
    // Application Secrets - Paste your values here
    PORT: "8080",
    DATABASE_URL: "",
    CLERK_SECRET_KEY: "",
    GOOGLE_CLIENT_ID: "",
    GOOGLE_CLIENT_SECRET: "",
    AWS_REGION: "",
    AWS_ACCESS_KEY_ID: "",
    AWS_SECRET_ACCESS_KEY: "",
    S3_BUCKET_NAME: "",
    GEMINI_API_KEY: "",
    QSTASH_URL: "",
    QSTASH_TOKEN: "",
    QSTASH_CURRENT_SIGNING_KEY: "",
    QSTASH_NEXT_SIGNING_KEY: "",
    ENCRYPTION_KEY: "",
    ENVIRONMENT: "production",
    NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY: "",
    NEXT_PUBLIC_API_URL: "https://api.follow.email",
    NEXT_PUBLIC_APP_URL: "https://follow.email"
};
