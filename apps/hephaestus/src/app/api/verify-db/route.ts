
import { NextResponse } from 'next/server';
import { db } from '@/lib/db';

export const dynamic = 'force-dynamic';

export async function GET() {
    try {
        const sshKeys = await db.query('SELECT * FROM ssh_keys');
        const sslCerts = await db.query('SELECT * FROM ssl_certificates');

        return NextResponse.json({
            ssh_keys: sshKeys.rows,
            ssl_certificates: sslCerts.rows
        });
    } catch (error: any) {
        return NextResponse.json({ error: error.message }, { status: 500 });
    }
}
