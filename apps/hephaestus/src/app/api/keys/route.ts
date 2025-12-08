
import { NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { getSSHKeys } from '@/lib/credentials';

export const dynamic = 'force-dynamic';

export async function GET() {
    try {
        const keys = await getSSHKeys();
        return NextResponse.json(keys);
    } catch (error: any) {
        console.error('Get Keys Error:', error);
        return NextResponse.json(
            { error: error.message || 'Failed to fetch keys' },
            { status: 500 }
        );
    }
}

export async function POST(request: Request) {
    try {
        // Mock POST for consistency, but we don't save to file in this demo mode
        const body = await request.json();
        const { name, public_key, private_key } = body;

        // Return a mock success response
        const newKey = {
            id: Math.floor(Math.random() * 1000) + 10,
            name,
            public_key,
            private_key,
            created_at: new Date().toISOString()
        };

        return NextResponse.json(newKey);
    } catch (error: any) {
        return NextResponse.json(
            { error: error.message || 'Failed to create key' },
            { status: 500 }
        );
    }
}
