import { NextResponse } from 'next/server';
import { AuthDB } from '@/lib/auth-db';
import bcrypt from 'bcryptjs';
import * as jose from 'jose';

export async function POST(request: Request) {
    try {
        const body = await request.json();
        const { email, password } = body;

        if (!email || !password) {
            return NextResponse.json({ error: 'Email and password are required' }, { status: 400 });
        }

        const user = await AuthDB.getUserByEmail(email);

        if (!user) {
            return NextResponse.json({ error: 'Invalid credentials' }, { status: 401 });
        }

        const isValid = await bcrypt.compare(password, user.password_hash);

        if (!isValid) {
            return NextResponse.json({ error: 'Invalid credentials' }, { status: 401 });
        }

        // Create JWT
        const secret = new TextEncoder().encode(process.env.JWT_SECRET || 'fallback-secret-key-change-me');
        const token = await new jose.SignJWT({ 'urn:example:claim': true, email: user.email, sub: user.id.toString() })
            .setProtectedHeader({ alg: 'HS256' })
            .setIssuedAt()
            .setExpirationTime('24h')
            .sign(secret);

        // Set Cookie
        const response = NextResponse.json({ success: true });
        response.cookies.set('session_token', token, {
            httpOnly: true,
            secure: process.env.NODE_ENV === 'production',
            sameSite: 'strict',
            maxAge: 60 * 60 * 24, // 1 day
            path: '/',
        });

        return response;
    } catch (error: any) {
        console.error('Login Error:', error);
        return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
    }
}
