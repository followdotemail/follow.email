import { db } from './db';
import bcrypt from 'bcryptjs';

export interface User {
    id: number;
    email: string;
    password_hash: string;
}

export const AuthDB = {
    async createUserTable() {
        const query = `
            CREATE TABLE IF NOT EXISTS provisioning_users (
                id SERIAL PRIMARY KEY,
                email VARCHAR(255) UNIQUE NOT NULL,
                password_hash VARCHAR(255) NOT NULL,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
            );
        `;
        await db.query(query);
    },

    async getUserByEmail(email: string): Promise<User | null> {
        const query = 'SELECT * FROM provisioning_users WHERE email = $1';
        const result = await db.query(query, [email]);
        return result.rows[0] || null;
    },

    async createUser(email: string, password: string) {
        const passwordHash = await bcrypt.hash(password, 10);
        const query = 'INSERT INTO provisioning_users (email, password_hash) VALUES ($1, $2) RETURNING id, email';
        try {
            const result = await db.query(query, [email, passwordHash]);
            return result.rows[0];
        } catch (error: any) {
            if (error.code === '23505') { // Unique violation
                throw new Error('User already exists');
            }
            throw error;
        }
    }
};
