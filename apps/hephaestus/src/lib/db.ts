
import { Pool } from 'pg';

if (!process.env.DATABASE_URL) {
    throw new Error('DATABASE_URL environment variable is missing. Please check your .env file or terminal environment.');
}

// Fix: Strip surrounding quotes if present (common issue in Windows/Next.js env loading)
const connectionString = process.env.DATABASE_URL.replace(/^"|"$/g, '').replace(/^'|'$/g, '');

const pool = new Pool({
    connectionString,
    // SSL is configured via the connection string query params (sslmode=require)
});

export const db = {
    query: (text: string, params?: any[]) => pool.query(text, params),
    pool,
};
