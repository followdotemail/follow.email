
import { NextResponse } from 'next/server';
import { ExcloudAPI } from '@/lib/external-apis';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
    try {
        const body = await request.json();
        const { vm_id, zone_id } = body;

        if (!vm_id) {
            return NextResponse.json(
                { error: 'Missing required field: vm_id' },
                { status: 400 }
            );
        }

        const instance = await ExcloudAPI.getInstanceStatus(
            parseInt(vm_id),
            zone_id ? parseInt(zone_id) : 1
        );

        if (!instance) {
            return NextResponse.json({ error: "Instance not found" }, { status: 404 });
        }

        const isReady = instance.state.toLowerCase() === 'running' || instance.state.toLowerCase() === 'active';

        return NextResponse.json({
            success: true,
            state: instance.state,
            ip_address: instance.public_ipv4,
            is_ready: isReady,
            data: instance
        });
    } catch (error: any) {
        console.error('Status Check Error:', error);
        return NextResponse.json(
            { error: error.message || 'Failed to check status' },
            { status: 500 }
        );
    }
}
