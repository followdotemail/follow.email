
import { Pool } from 'pg';

// Lazy initialization of the pool
let pool: Pool | null = null;

function getPool(): Pool {
    if (!pool) {
        if (!process.env.DATABASE_URL) {
            throw new Error('DATABASE_URL environment variable is missing. Please check your .env file or terminal environment.');
        }
        // Fix: Strip surrounding quotes if present
        const connectionString = process.env.DATABASE_URL.replace(/^"|"$/g, '').replace(/^'|'$/g, '');
        pool = new Pool({
            connectionString,
        });
    }
    return pool;
}

export const db = {
    query: (text: string, params?: any[]) => getPool().query(text, params),
};
