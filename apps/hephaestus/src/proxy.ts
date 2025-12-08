import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import * as jose from 'jose';

export async function proxy(request: NextRequest) {
    const path = request.nextUrl.pathname;

    // 1. Define public paths that don't need authentication
    const publicPaths = [
        '/login',
        '/api/auth/login',
        '/_next',
        '/favicon.ico',
    ];

    // Check if the path starts with any of the public paths
    const isPublicPath = publicPaths.some((pp) => path.startsWith(pp));

    if (isPublicPath) {
        return NextResponse.next();
    }

    // 2. Check for session token
    const token = request.cookies.get('session_token')?.value;

    if (!token) {
        // Redirect to login if no token
        return NextResponse.redirect(new URL('/login', request.url));
    }

    try {
        // 3. Verify JWT
        const secret = new TextEncoder().encode(process.env.JWT_SECRET || 'fallback-secret-key-change-me');
        await jose.jwtVerify(token, secret);

        // Token is valid, proceed
        return NextResponse.next();
    } catch (error) {
        // Token invalid or expired
        console.error("JWT Verification failed:", error);
        return NextResponse.redirect(new URL('/login', request.url));
    }
}

export const config = {
    matcher: [
        /*
         * Match all request paths except for the ones starting with:
         * - api (except for specific protected api routes if desired, but we protect all by default here unless excluded above)
         * - _next/static (static files)
         * - _next/image (image optimization files)
         * - favicon.ico (favicon file)
         */
        '/((?!_next/static|_next/image|favicon.ico).*)',
    ],
};
