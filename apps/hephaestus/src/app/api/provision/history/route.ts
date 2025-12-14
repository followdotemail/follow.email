
import { NextResponse } from 'next/server';
import { ExcloudAPI } from '@/lib/external-apis';

export const dynamic = 'force-dynamic';

export async function GET(req: Request) {
    try {
        const { searchParams } = new URL(req.url);
        const states = searchParams.getAll('states'); // Can be multiple
        const created_after = searchParams.get('created_after') || undefined;
        const created_before = searchParams.get('created_before') || undefined;

        const data = await ExcloudAPI.getProvisioningHistory({
            states: states.length > 0 ? states : undefined,
            created_after,
            created_before
        });
        return NextResponse.json(data);
    } catch (error: any) {
        console.error('Provisioning History Error:', error);

        const status = error.response?.status || 500;
        const message = error.response?.data?.error || error.message || 'Unknown error';

        return NextResponse.json(
            { error: message },
            { status: status }
        );
    }
}
